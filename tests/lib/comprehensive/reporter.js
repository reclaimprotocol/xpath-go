'use strict';

const fs = require('fs');
const path = require('path');

function printTestResult(testCase, result, testNumber, total, options, log = console.log) {
    if (options.failedOnly && result.passed) return;

    const extractionModeIcon = testCase.extractionMode === 'content'
        ? '📄'
        : testCase.extractionMode === 'full' ? '🏷️' : '';
    log(`[${testNumber}/${total}] ${testCase.name} ${testCase.suite === 'extended' ? '🆕' : ''} ${extractionModeIcon}`);
    log(`XPath: ${testCase.xpath}`);

    if (testCase.extractionMode) {
        const mode = testCase.extractionMode === 'content' ? 'Content-Only Mode' : 'Full Element Mode';
        log(`Mode: ${mode}`);
    }
    if (testCase.filepath) log(`📁 HTML file: ${testCase.filepath}`);
    if (options.verbose) {
        if (testCase.html && !testCase.filepath) log(`HTML: ${testCase.html}`);
        else if (testCase.filepath) log(`HTML loaded from file: ${testCase.filepath}`);
    }

    if (result.passed) printPassingResult(result, options, log);
    else printFailingResult(result, options, log);
}

function printPassingResult(result, options, log) {
    log('✅ PASS - Results match perfectly');
    log(`   Found ${result.jsResult.count} matching nodes`);

    if (options.showData && result.jsResult.count > 0) {
        log('\n   📊 JavaScript Results:');
        log(JSON.stringify(result.jsResult.results, null, 2));
        log('\n   📊 Go Results:');
        log(JSON.stringify(result.goResult.results, null, 2));
    } else if (options.verbose && result.jsResult.count > 0) {
        log('   Results:');
        printPreview(result.jsResult, log);
    }
    log();
}

function printFailingResult(result, options, log) {
    log('❌ FAIL - Results differ');
    log(`   Reason: ${result.comparison.reason}`);
    if (options.showData) {
        log(`\n   📊 JavaScript Results (${result.jsResult.count}):`);
        log(JSON.stringify(result.jsResult.results, null, 2));
        log(`\n   📊 Go Results (${result.goResult.count}):`);
        log(JSON.stringify(result.goResult.results, null, 2));
    } else if (options.verbose || options.trace) {
        log(`   JavaScript Results (${result.jsResult.count}):`);
        printPreview(result.jsResult, log, false);
        log(`   Go Results (${result.goResult.count}):`);
        printPreview(result.goResult, log, false);
    }
    log();
}

function printPreview(result, log, showRemainder = true) {
    result.results.slice(0, 3).forEach((node, index) => {
        const text = node.textContent ? `: '${node.textContent}'` : '';
        log(`     ${index + 1}. <${node.nodeName}>${text} [pos: ${node.startLocation}-${node.endLocation}]`);
    });
    if (showRemainder && result.count > 3) {
        log(`     ... and ${result.count - 3} more`);
    }
}

function generateReport(results, testsDirectory, log = console.log) {
    results.compatibility = (results.passed / results.total * 100).toFixed(1);
    log('🎯 COMPREHENSIVE COMPATIBILITY REPORT');
    log('=====================================');
    log(`Total Tests: ${results.total}`);
    log(`Passed: ${results.passed}`);
    log(`Failed: ${results.failed}`);
    log(`Overall Compatibility: ${results.compatibility}%\\n`);

    const originalResults = results.details.filter(result => result.suite === 'original');
    const extendedResults = results.details.filter(result => result.suite === 'extended');
    log('📊 DETAILED BREAKDOWN');
    log('=====================');
    printSuiteSummary('Original', originalResults, log);
    printSuiteSummary('Extended', extendedResults, log);
    printCategoryBreakdown(extendedResults, log);
    printExtractionBreakdown(results.details, log);
    printFailures(results, log);

    const reportPath = path.join(testsDirectory, 'comprehensive_compatibility_report.json');
    fs.writeFileSync(reportPath, JSON.stringify(results, null, 2));
    log(`\\n📊 Detailed report saved to: ${reportPath}`);
}

function printSuiteSummary(label, suiteResults, log) {
    const passed = suiteResults.filter(result => result.passed).length;
    const percentage = (passed / suiteResults.length * 100).toFixed(1);
    log(`${label} Suite: ${passed}/${suiteResults.length} (${percentage}%)`);
}

function printCategoryBreakdown(extendedResults, log) {
    if (extendedResults.length === 0) return;
    log('\\n🏷️ CATEGORY BREAKDOWN (Extended Tests)');
    log('======================================');
    const categories = {};
    for (const result of extendedResults) {
        const category = result.category || 'uncategorized';
        categories[category] ||= { total: 0, passed: 0 };
        categories[category].total++;
        if (result.passed) categories[category].passed++;
    }
    for (const category of Object.keys(categories).sort()) {
        const { total, passed } = categories[category];
        const percentage = (passed / total * 100).toFixed(1);
        log(`${category.padEnd(20)}: ${passed}/${total} (${percentage}%)`);
    }
}

function printExtractionBreakdown(details, log) {
    const extractionTests = details.filter(result => result.extractionMode);
    if (extractionTests.length === 0) return;
    log('\\n📄 CONTENTS-ONLY EXTRACTION BREAKDOWN');
    log('======================================');
    printModeSummary('Full Element Mode  ', extractionTests, 'full', log);
    printModeSummary('Content-Only Mode  ', extractionTests, 'content', log);
}

function printModeSummary(label, results, mode, log) {
    const modeResults = results.filter(result => result.extractionMode === mode);
    if (modeResults.length === 0) return;
    const passed = modeResults.filter(result => result.passed).length;
    const percentage = (passed / modeResults.length * 100).toFixed(1);
    log(`${label}: ${passed}/${modeResults.length} (${percentage}%)`);
}

function printFailures(results, log) {
    if (results.failed === 0) {
        log('\\n🎉 PERFECT COMPATIBILITY ACHIEVED! 🎉');
        return;
    }
    log('\\n❌ FAILING TESTS');
    log('=================');
    results.details.filter(result => !result.passed).forEach((result, index) => {
        log(`${index + 1}. ${result.name} ${result.suite === 'extended' ? '🆕' : ''}`);
        log(`   XPath: ${result.xpath}`);
        log(`   Reason: ${result.comparison.reason}`);
        log();
    });
    log(`🔧 ${results.failed} tests need attention for 100% compatibility`);
}

module.exports = { generateReport, printTestResult };
