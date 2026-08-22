'use strict';

const { compareResults } = require('./compare_results');
const { GoOracle } = require('./go_oracle');
const { JavaScriptOracle } = require('./jsdom_oracle');
const { generateReport, printTestResult } = require('./reporter');
const { getHtmlContent, loadTestCases } = require('./test_cases');

class ComprehensiveXPathTester {
    constructor(testsDirectory, options = {}, dependencies = {}) {
        this.testsDirectory = testsDirectory;
        this.options = {
            trace: options.trace || false,
            singleTest: options.singleTest || null,
            verbose: options.verbose || false,
            failedOnly: options.failedOnly || false,
            showData: options.showData || false
        };
        this.log = dependencies.log || console.log;
        this.javascriptOracle = dependencies.javascriptOracle || new JavaScriptOracle();
        this.goOracle = dependencies.goOracle || new GoOracle(testsDirectory, this.options);
        this.results = emptyResults();
    }

    async run() {
        this.log('🧪 Starting Comprehensive XPath Compatibility Testing');
        this.log('=====================================================\\n');
        const testCases = loadTestCases(
            this.testsDirectory,
            this.options.singleTest,
            this.log
        );
        this.log();
        this.results.total = testCases.length;

        try {
            for (let index = 0; index < testCases.length; index++) {
                const testCase = testCases[index];
                const result = await this.runSingleTest(testCase);
                this.results.details.push(result);
                if (result.passed) this.results.passed++;
                else this.results.failed++;
                printTestResult(
                    testCase,
                    result,
                    index + 1,
                    testCases.length,
                    this.options,
                    this.log
                );
            }
            generateReport(this.results, this.testsDirectory, this.log);
            return this.results;
        } finally {
            this.goOracle.dispose();
        }
    }

    async runSingleTest(testCase) {
        const contentsOnly = testCase.extractionMode === 'content';
        const html = getHtmlContent(testCase, this.testsDirectory);
        const jsResult = this.javascriptOracle.evaluate(testCase.xpath, html, contentsOnly, testCase);
        const goResult = this.goOracle.evaluate(testCase.xpath, html, contentsOnly, testCase);
        const comparison = compareResults(jsResult, goResult);

        return {
            name: testCase.name,
            xpath: testCase.xpath,
            suite: testCase.suite,
            category: testCase.category,
            extractionMode: testCase.extractionMode,
            passed: comparison.match,
            jsResult,
            goResult,
            comparison
        };
    }
}

function emptyResults() {
    return {
        total: 0,
        passed: 0,
        failed: 0,
        compatibility: 0,
        details: []
    };
}

module.exports = { ComprehensiveXPathTester, emptyResults };
