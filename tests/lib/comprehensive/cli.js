'use strict';

const DEFAULT_OPTIONS = Object.freeze({
    trace: false,
    singleTest: null,
    verbose: false,
    failedOnly: false,
    showData: false,
    help: false
});

function parseArgs(args = process.argv.slice(2)) {
    const options = { ...DEFAULT_OPTIONS };
    for (let index = 0; index < args.length; index++) {
        const argument = args[index];
        switch (argument) {
            case '--trace':
            case '-t':
                options.trace = true;
                break;
            case '--verbose':
            case '-v':
                options.verbose = true;
                break;
            case '--failed-only':
            case '-f':
                options.failedOnly = true;
                break;
            case '--show-data':
            case '-d':
                options.showData = true;
                break;
            case '--test':
            case '-s':
                if (index + 1 >= args.length) {
                    throw new Error('--test requires a test number or name');
                }
                options.singleTest = args[++index];
                break;
            case '--help':
            case '-h':
                options.help = true;
                break;
            default:
                if (argument.startsWith('-')) throw new Error(`Unknown option: ${argument}`);
                options.singleTest = argument;
        }
    }
    return options;
}

function showHelp(log = console.log) {
    log(`
🧪 XPath Compatibility Test Suite

Usage: node comprehensive_compare.js [options] [test_identifier]

Options:
  --trace, -t           Enable Go XPath trace mode for debugging
  --verbose, -v         Show detailed test output and results
  --show-data, -d       Display returned data from both JS and Go implementations
  --failed-only, -f     Only show failing tests
  --test, -s <id>       Run single test by number or name pattern
  --help, -h            Show this help message

Examples:
  node comprehensive_compare.js                    # Run all tests
  node comprehensive_compare.js --trace           # Run all tests with trace
  node comprehensive_compare.js --failed-only     # Show only failing tests
  node comprehensive_compare.js --show-data       # Display full result data from both implementations
  node comprehensive_compare.js --test 62         # Run test #62
  node comprehensive_compare.js --test position   # Run tests matching "position"
  node comprehensive_compare.js --test 1 -d       # Run test #1 with full data output
  node comprehensive_compare.js -t -v --test 62   # Run test #62 with trace and verbose
  node comprehensive_compare.js substring         # Run tests matching "substring"

Trace mode shows detailed XPath evaluation steps from the Go implementation.
Single test mode allows focusing on specific failing cases for debugging.
`);
}

function announceOptions(options, log = console.log) {
    if (options.trace) log('🔍 Trace mode enabled - will show detailed XPath evaluation steps');
    if (options.showData) log('📊 Show data mode enabled - will display full result data from both implementations');
    if (options.singleTest) log(`🎯 Single test mode: "${options.singleTest}"`);
}

module.exports = { announceOptions, parseArgs, showHelp };
