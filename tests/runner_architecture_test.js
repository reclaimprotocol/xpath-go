'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { parseArgs } = require('./lib/comprehensive/cli');
const { compareResults } = require('./lib/comprehensive/compare_results');
const { JavaScriptOracle } = require('./lib/comprehensive/jsdom_oracle');
const { loadTestCases } = require('./lib/comprehensive/test_cases');

const TESTS_DIRECTORY = __dirname;
const RUNNER_DIRECTORY = path.join(TESTS_DIRECTORY, 'lib', 'comprehensive');

test('compatibility fixtures retain their established suite boundaries', () => {
    const fixtures = loadTestCases(TESTS_DIRECTORY, null, () => {});
    assert.equal(fixtures.length, 888);
    assert.equal(fixtures.filter(testCase => testCase.suite === 'original').length, 37);
    assert.equal(fixtures.filter(testCase => testCase.suite === 'extended').length, 851);
});

test('the entry point and runner modules stay focused', () => {
    assert.ok(lineCount(path.join(TESTS_DIRECTORY, 'comprehensive_compare.js')) <= 50);
    for (const filename of fs.readdirSync(RUNNER_DIRECTORY)) {
        if (!filename.endsWith('.js')) continue;
        assert.ok(lineCount(path.join(RUNNER_DIRECTORY, filename)) <= 250, `${filename} grew too large`);
    }
});

test('CLI parsing supports aliases, selectors, and invalid options', () => {
    assert.deepEqual(parseArgs(['-t', '-v', '-f', '-d', '-s', 'position']), {
        trace: true,
        singleTest: 'position',
        verbose: true,
        failedOnly: true,
        showData: true,
        help: false
    });
    assert.equal(parseArgs(['62']).singleTest, '62');
    assert.throws(() => parseArgs(['--test']), /requires a test number or name/);
    assert.throws(() => parseArgs(['--unknown']), /Unknown option/);
});

test('result comparison reports the first material difference', () => {
    const jsResult = successfulResult({
        nodeName: 'p', nodeType: 1, textContent: 'hello', startLocation: 0, endLocation: 12
    });
    const matching = successfulResult({
        nodeName: 'p', nodeType: 1, textContent: 'hello', startLocation: 0, endLocation: 12
    });
    assert.deepEqual(compareResults(jsResult, matching), {
        match: true,
        reason: 'Perfect match',
        jsResults: jsResult.results,
        goResults: matching.results
    });

    const different = successfulResult({
        nodeName: 'p', nodeType: 1, textContent: 'goodbye', startLocation: 0, endLocation: 12
    });
    assert.match(compareResults(jsResult, different).reason, /^Text content mismatch/);
});

test('jsdom oracle converts UTF-16 source locations to UTF-8 byte offsets', () => {
    const oracle = new JavaScriptOracle();
    const result = oracle.evaluate('//p', '<p>💡x</p>');
    assert.equal(result.success, true);
    assert.equal(result.results[0].startLocation, 0);
    assert.equal(result.results[0].endLocation, 12);
    assert.equal(result.results[0].value, '<p>💡x</p>');

    const contentResult = oracle.evaluate('//p', '<p>💡x</p>', true);
    assert.equal(contentResult.results[0].startLocation, 3);
    assert.equal(contentResult.results[0].endLocation, 8);
});

function successfulResult(node) {
    return { success: true, results: [node], count: 1, error: null };
}

function lineCount(filename) {
    return fs.readFileSync(filename, 'utf8').split('\n').length;
}
