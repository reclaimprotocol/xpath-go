#!/usr/bin/env node
'use strict';

const { announceOptions, parseArgs, showHelp } = require('./lib/comprehensive/cli');
const { ComprehensiveXPathTester } = require('./lib/comprehensive/runner');

async function main() {
    let options;
    try {
        options = parseArgs();
    } catch (error) {
        console.error(`Error: ${error.message}`);
        process.exitCode = 1;
        return;
    }

    if (options.help) {
        showHelp();
        return;
    }

    announceOptions(options);
    const tester = new ComprehensiveXPathTester(__dirname, options);
    await tester.run();
}

main().catch(error => {
    console.error('Test runner error:', error);
    process.exitCode = 1;
});
