package main
import "fmt"

func main() {
    expr := "//div"
    for i := 0; i < len(expr); i++ {
        c := expr[i]
        fmt.Printf("pos %d: char='%c', isLetter=%v, isNameStart=%v\n", 
            i, c, isLetter(c), isNameStart(c))
    }
}

func isLetter(c byte) bool {
    return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNameStart(c byte) bool {
    return isLetter(c) || c == '_'
}
