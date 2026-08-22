'use strict';

const fs = require('fs');
const path = require('path');

const SUITES = [
    { name: 'original', file: 'testcases.json' },
    { name: 'extended', file: 'extended_testcases.json' }
];

function loadTestCases(testsDirectory, singleTest = null, log = console.log) {
    const sharedDirectory = path.join(testsDirectory, 'shared');
    const allTestCases = SUITES.flatMap(suite => {
        const fixturePath = path.join(sharedDirectory, suite.file);
        if (!fs.existsSync(fixturePath)) return [];

        const fixtures = JSON.parse(fs.readFileSync(fixturePath, 'utf8'));
        return fixtures.map(testCase => ({ ...testCase, suite: suite.name }));
    });

    if (singleTest) {
        return selectTestCases(allTestCases, singleTest, log);
    }

    log(`\n📚 Loaded ${allTestCases.length} test cases:`);
    for (const suite of SUITES) {
        const count = allTestCases.filter(testCase => testCase.suite === suite.name).length;
        const label = suite.name[0].toUpperCase() + suite.name.slice(1);
        log(`   • ${label} suite: ${count} tests`);
    }
    log();
    return allTestCases;
}

function selectTestCases(allTestCases, identifier, log) {
    const testIndex = Number.parseInt(identifier, 10) - 1;
    const testName = identifier.toLowerCase();

    if (!Number.isNaN(testIndex) && testIndex >= 0 && testIndex < allTestCases.length) {
        const selected = [allTestCases[testIndex]];
        log(`\n🎯 Running single test #${identifier}: ${selected[0].name}`);
        return selected;
    }

    const selected = allTestCases.filter(testCase =>
        testCase.name.toLowerCase().includes(testName) ||
        testCase.xpath.toLowerCase().includes(testName)
    );
    if (selected.length === 0) {
        throw new Error(`No tests found matching "${identifier}"`);
    }

    log(`\n🎯 Found ${selected.length} test(s) matching "${identifier}":`);
    selected.forEach((testCase, index) => log(`   ${index + 1}. ${testCase.name}`));
    return selected;
}

function resolveFixturePath(testsDirectory, fixturePath) {
    if (path.isAbsolute(fixturePath)) return fixturePath;

    const relativePath = fixturePath.startsWith('tests/')
        ? fixturePath.substring(6)
        : fixturePath;
    return path.resolve(testsDirectory, relativePath);
}

function getHtmlFilePath(testCase, testsDirectory) {
    if (!testCase.filepath) return null;

    const fullPath = resolveFixturePath(testsDirectory, testCase.filepath);
    if (!fs.existsSync(fullPath)) {
        throw new Error(`HTML file not found: ${fullPath}`);
    }
    return fullPath;
}

function getHtmlContent(testCase, testsDirectory) {
    if (testCase.htmlBase64) {
        return Buffer.from(testCase.htmlBase64, 'base64');
    }

    const htmlFilePath = getHtmlFilePath(testCase, testsDirectory);
    if (htmlFilePath) {
        return testCase.charset
            ? fs.readFileSync(htmlFilePath)
            : fs.readFileSync(htmlFilePath, 'utf8');
    }

    if (!Object.prototype.hasOwnProperty.call(testCase, 'html')) {
        throw new Error(`Test case '${testCase.name}' has neither 'html' nor 'filepath' field`);
    }
    return testCase.html;
}

module.exports = {
    getHtmlContent,
    getHtmlFilePath,
    loadTestCases,
    resolveFixturePath
};
