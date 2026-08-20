package xpath

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// sourceSegment records the raw byte range that produced a contiguous UTF-8
// range in the parser input. Keeping segments per decoded character/chunk is
// substantially smaller than a decoded-byte-to-raw-byte table for non-ASCII
// documents while retaining exact locations at every source boundary.
type sourceSegment struct {
	decodedStart int
	decodedEnd   int
	rawStart     int
	rawEnd       int
	decodedUnit  int
	rawUnit      int
}

type sourceMapper struct {
	segments      []sourceSegment
	decodedLength int
	rawLength     int
}

// decodeForXPath creates the logical UTF-8 source used by the HTML parser.
// The caller retains raw for public result slicing. An empty charset is the
// compatibility mode: malformed UTF-8 is tolerated and replaced with U+FFFD.
func decodeForXPath(raw []byte, charset string) (string, sourceMapper, error) {
	if err := validatePublicHTMLInput(raw); err != nil {
		return "", sourceMapper{}, err
	}

	label := strings.TrimSpace(charset)
	if label == "" {
		decoded, mapper := decodeUTF8Lossy(raw)
		return decoded, mapper, nil
	}

	enc, err := htmlindex.Get(label)
	if err != nil {
		return "", sourceMapper{}, fmt.Errorf("unsupported charset %q: %w", charset, err)
	}
	canonical, err := htmlindex.Name(enc)
	if err != nil {
		return "", sourceMapper{}, fmt.Errorf("unsupported charset %q: %w", charset, err)
	}
	if canonical == "utf-8" {
		decoded, mapper := decodeUTF8Lossy(raw)
		return decoded, mapper, nil
	}
	decodeRaw := raw
	rawPrefix := 0
	if canonical == "utf-16le" && len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xfe {
		decodeRaw = raw[2:]
		rawPrefix = 2
	} else if canonical == "utf-16be" && len(raw) >= 2 && raw[0] == 0xfe && raw[1] == 0xff {
		decodeRaw = raw[2:]
		rawPrefix = 2
	}
	decoded, mapper, err := decodeWithEncoding(decodeRaw, enc)
	if err != nil {
		return "", sourceMapper{}, err
	}
	if rawPrefix > 0 {
		for index := range mapper.segments {
			mapper.segments[index].rawStart += rawPrefix
			mapper.segments[index].rawEnd += rawPrefix
		}
		mapper.rawLength = len(raw)
	}
	return decoded, mapper, nil
}

// validatePublicHTMLInput deliberately differs from utils.HTMLParser.Parse:
// response bytes may be a legacy encoding or malformed UTF-8. Only a gzip
// signature is meaningful before decoding; bytes that look like ASCII controls
// can be ordinary code-unit bytes in encodings such as UTF-16. The strict HTML
// parser validates actual control characters after decoding.
func validatePublicHTMLInput(raw []byte) error {
	if len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		return fmt.Errorf("binary input is not supported")
	}
	return nil
}

func decodeUTF8Lossy(raw []byte) (string, sourceMapper) {
	var decoded strings.Builder
	decoded.Grow(len(raw))
	// Most HTML is one or a handful of affine runs. Grow on demand instead of
	// reserving in proportion to the response body: sourceSegment is large
	// enough that len(raw)/2 capacity can multiply peak memory many times over.
	var segments []sourceSegment

	rawStart := 0
	if len(raw) >= 3 && raw[0] == 0xef && raw[1] == 0xbb && raw[2] == 0xbf {
		rawStart = 3
	}
	for rawStart < len(raw) {
		r, size := utf8.DecodeRune(raw[rawStart:])
		if r == utf8.RuneError && size == 1 {
			// DecodeRune deliberately consumes one malformed byte at a time,
			// matching the source-boundary behavior required for offset mapping.
			r = '\uFFFD'
		}
		decodedStart := decoded.Len()
		decoded.WriteRune(r)
		segments = appendSourceSegment(segments, decodedStart, decoded.Len(), rawStart, rawStart+size)
		rawStart += size
	}

	return decoded.String(), sourceMapper{
		segments:      segments,
		decodedLength: decoded.Len(),
		rawLength:     len(raw),
	}
}

// decodeWithEncoding streams one input byte at a time. A decoder may need to
// look ahead (Shift_JIS, UTF-16 and stateful encodings), so a segment spans all
// raw bytes consumed since the preceding output. This also handles zero-output
// sequences such as an encoding preamble without allocating a per-byte map.
func decodeWithEncoding(raw []byte, enc encoding.Encoding) (string, sourceMapper, error) {
	decoder := enc.NewDecoder()
	output := make([]byte, utf8.UTFMax)
	var decoded strings.Builder
	decoded.Grow(len(raw))
	// appendSourceSegment coalesces ordinary fixed-width runs, so input length
	// is not a useful estimate of the final segment count.
	var segments []sourceSegment

	availableEnd := 0
	unconsumedStart := 0
	spanStart := 0
	spanActive := false

	for availableEnd < len(raw) {
		availableEnd++
		if !spanActive {
			spanStart = unconsumedStart
			spanActive = true
		}

		for {
			atEOF := availableEnd == len(raw)
			nDst, nSrc, err := transformOneRune(decoder, output, raw[unconsumedStart:availableEnd], atEOF)
			unconsumedStart += nSrc

			if nDst > 0 {
				decodedStart := decoded.Len()
				decoded.Write(output[:nDst])
				segments = appendSourceSegment(segments, decodedStart, decoded.Len(), spanStart, unconsumedStart)
				spanStart = unconsumedStart
				spanActive = false
			}

			switch err {
			case nil:
				// A successful zero-output transform (for example, a BOM) is
				// not part of the next character's source range.
				if nDst == 0 {
					spanStart = unconsumedStart
					spanActive = false
				}
			case transform.ErrShortSrc:
				if atEOF {
					// x/text decoders normally emit U+FFFD at EOF. Preserve a
					// tolerant public path if a decoder instead reports short input.
					if spanActive {
						decodedStart := decoded.Len()
						decoded.WriteRune('\uFFFD')
						segments = appendSourceSegment(segments, decodedStart, decoded.Len(), spanStart, len(raw))
					}
					availableEnd = len(raw)
					unconsumedStart = len(raw)
					spanActive = false
				}
				// More input is needed. If the decoder consumed bytes into
				// internal state, the next loop supplies only the next byte;
				// otherwise it retains the unconsumed prefix for look-ahead.
			case transform.ErrShortDst:
				// transformOneRune deliberately limits the destination to one
				// UTF-8 rune. If the decoder filled it and still has source
				// remaining, continue with the same source window so each output
				// rune gets its own raw span.
				if nDst > 0 {
					if unconsumedStart < availableEnd {
						spanStart = unconsumedStart
						spanActive = true
						continue
					}
					spanActive = false
					break
				}
				return "", sourceMapper{}, fmt.Errorf("charset decoder output buffer exhausted")
			default:
				return "", sourceMapper{}, fmt.Errorf("charset decoding failed: %w", err)
			}

			// Transform has either consumed all currently available data or
			// needs more source. It must not be called again until another raw
			// byte is appended; doing so would spin on ErrShortSrc.
			break
		}
	}

	return decoded.String(), sourceMapper{
		segments:      segments,
		decodedLength: decoded.Len(),
		rawLength:     len(raw),
	}, nil
}

// transformOneRune bounds the destination to the minimum width needed by the
// first UTF-8 rune. x/text transformers return ErrShortDst without consuming
// source when the first rune does not fit, so retrying with a wider destination
// still leaves exactly one output rune per call. This prevents malformed input
// such as a Shift_JIS lead byte followed by '<' from collapsing two decoded
// boundaries into one raw span.
func transformOneRune(decoder *encoding.Decoder, output []byte, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for width := 1; width <= utf8.UTFMax; width++ {
		nDst, nSrc, err = decoder.Transform(output[:width], src, atEOF)
		if err != transform.ErrShortDst || nDst > 0 {
			return nDst, nSrc, err
		}
	}
	return nDst, nSrc, err
}

// appendSourceSegment coalesces adjacent units with identical raw/decoded
// widths. Long ASCII (and fixed-width legacy-encoding) runs are consequently
// stored as one affine range instead of an entry for every source byte.
func appendSourceSegment(segments []sourceSegment, decodedStart, decodedEnd, rawStart, rawEnd int) []sourceSegment {
	decodedUnit := decodedEnd - decodedStart
	rawUnit := rawEnd - rawStart
	if decodedUnit <= 0 || rawUnit <= 0 {
		return segments
	}
	if len(segments) > 0 {
		previous := &segments[len(segments)-1]
		if previous.decodedEnd == decodedStart && previous.rawEnd == rawStart &&
			previous.decodedUnit == decodedUnit && previous.rawUnit == rawUnit {
			previous.decodedEnd = decodedEnd
			previous.rawEnd = rawEnd
			return segments
		}
	}
	return append(segments, sourceSegment{
		decodedStart: decodedStart,
		decodedEnd:   decodedEnd,
		rawStart:     rawStart,
		rawEnd:       rawEnd,
		decodedUnit:  decodedUnit,
		rawUnit:      rawUnit,
	})
}

// rawOffsetLeft returns the raw boundary immediately before any zero-output
// source bytes at decodedOffset. Offsets inside a decoded rune conservatively
// map to the start of that rune's raw unit.
func (m sourceMapper) rawOffsetLeft(decodedOffset int) int {
	if decodedOffset <= 0 {
		return 0
	}
	if len(m.segments) == 0 {
		return 0
	}
	if decodedOffset > m.decodedLength {
		return m.rawLength
	}
	index := sort.Search(len(m.segments), func(i int) bool {
		return m.segments[i].decodedEnd >= decodedOffset
	})
	if index == len(m.segments) {
		return m.segments[len(m.segments)-1].rawEnd
	}
	segment := m.segments[index]
	if decodedOffset <= segment.decodedStart {
		if index > 0 && m.segments[index-1].decodedEnd == decodedOffset {
			return m.segments[index-1].rawEnd
		}
		return segment.rawStart
	}
	unitIndex := (decodedOffset - segment.decodedStart) / segment.decodedUnit
	return segment.rawStart + unitIndex*segment.rawUnit
}

// rawOffsetRight returns the raw boundary immediately after any zero-output
// source bytes at decodedOffset. This is the historical rawOffset behavior.
func (m sourceMapper) rawOffsetRight(decodedOffset int) int {
	if decodedOffset < 0 {
		return 0
	}
	if decodedOffset >= m.decodedLength {
		return m.rawLength
	}
	if len(m.segments) == 0 {
		return m.rawLength
	}
	index := sort.Search(len(m.segments), func(i int) bool {
		return m.segments[i].decodedEnd > decodedOffset
	})
	if index == len(m.segments) {
		return m.rawLength
	}
	segment := m.segments[index]
	if decodedOffset <= segment.decodedStart {
		return segment.rawStart
	}
	unitIndex := (decodedOffset - segment.decodedStart) / segment.decodedUnit
	return segment.rawStart + unitIndex*segment.rawUnit
}

// rawOffset retains the right-biased mapping used by callers that ask for a
// single raw boundary. Location remapping below selects a side per field.
func (m sourceMapper) rawOffset(decodedOffset int) int {
	return m.rawOffsetRight(decodedOffset)
}

// remapDocumentLocations translates locations after parsing the logical UTF-8
// view. Synthetic parser nodes deliberately retain their zero locations.
func (m sourceMapper) remapDocumentLocations(root *types.Node) {
	seen := make(map[*types.Node]struct{})
	var visit func(*types.Node)
	visit = func(node *types.Node) {
		if node == nil {
			return
		}
		if _, ok := seen[node]; ok {
			return
		}
		seen[node] = struct{}{}

		if node.Type == types.DocumentNode {
			node.StartPos = 0
			node.EndPos = m.rawLength
			node.SourceLength = m.rawLength
		} else if node.StartLine != 0 {
			// Stateful encodings can consume bytes without emitting Unicode, so
			// one decoded boundary can represent a raw interval. Full node ranges
			// exclude state bytes between adjacent nodes; content ranges include
			// all bytes physically located between the opening and closing tags.
			decodedStart := node.StartPos
			decodedEnd := node.EndPos
			decodedContentStart := node.ContentStart
			decodedContentEnd := node.ContentEnd
			node.StartPos = m.rawOffsetRight(decodedStart)
			node.EndPos = m.rawOffsetLeft(decodedEnd)
			if node.Type == types.ElementNode {
				node.ContentStart = m.rawOffsetLeft(decodedContentStart)
				node.ContentEnd = m.rawOffsetRight(decodedContentEnd)
			}
		}

		for _, child := range node.Children {
			visit(child)
		}
		visit(node.TemplateContent)
	}
	visit(root)
}
