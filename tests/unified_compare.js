#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const { JSDOM } = require('jsdom');

// Unified XPath compatibility tester - combines both core and extended test suites
class UnifiedXPathTester {
    constructor() {
        this.allTestCases = [];
        this.results = {
            total: 0,
            passed: 0,
            failed: 0,
            compatibility: 0,
            details: []
        };
    }

    loadAllTestCases() {
        // Load core test cases (original 37 tests)
        const corePath = path.join(__dirname, 'shared', 'testcases.json');
        if (fs.existsSync(corePath)) {
            const core = JSON.parse(fs.readFileSync(corePath, 'utf8'));
            this.allTestCases = this.allTestCases.concat(core.map(tc => ({...tc, suite: 'core'})));
            console.log(`📦 Loaded ${core.length} core test cases`);
        }

        // Load extended test cases (additional tests for comprehensive coverage)
        const extendedPath = path.join(__dirname, 'shared', 'extended_testcases.json');
        if (fs.existsSync(extendedPath)) {
            const extended = JSON.parse(fs.readFileSync(extendedPath, 'utf8'));
            this.allTestCases = this.allTestCases.concat(extended.map(tc => ({...tc, suite: 'extended'})));
            console.log(`📦 Loaded ${extended.length} extended test cases`);
        }

        console.log(`📊 Total test cases: ${this.allTestCases.length}`);
        return this.allTestCases.length;
    }

    async compareXPathResults(testCase) {
        try {
            // Get JavaScript XPath results (reference)
            const jsResults = await this.getJavaScriptResults(testCase.html, testCase.xpath);
            
            // Get Go XPath results (our implementation)
            const goResults = await this.getGoResults(testCase.html, testCase.xpath);
            
            // Compare results
            const isMatch = this.compareResults(jsResults, goResults);
            
            return {
                testCase,
                jsResults,
                goResults,
                match: isMatch,
                jsCount: jsResults.length,
                goCount: goResults.length
            };
        } catch (error) {
            return {
                testCase,
                error: error.message,
                match: false,
                jsCount: 0,
                goCount: 0
            };
        }
    }

    async getJavaScriptResults(html, xpath) {
        const dom = new JSDOM(html);
        const document = dom.window.document;
        
        try {
            const result = document.evaluate(
                xpath,
                document,
                null,
                dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
                null
            );
            
            const nodes = [];
            for (let i = 0; i < result.snapshotLength; i++) {
                const node = result.snapshotItem(i);
                nodes.push({
                    nodeName: node.nodeName.toLowerCase(),
                    textContent: node.textContent ? node.textContent.trim() : '',
                    attributes: this.getNodeAttributes(node)
                });
            }
            
            return nodes;
        } catch (error) {
            throw new Error(`JavaScript XPath error: ${error.message}`);
        }
    }

    async getGoResults(html, xpath) {
        try {
            // Create temporary files
            const timestamp = Date.now() + '_' + Math.random().toString(36).substr(2, 9);
            const htmlFile = path.join(__dirname, `temp_html_${timestamp}.html`);
            const xpathFile = path.join(__dirname, `temp_xpath_${timestamp}.txt`);
            
            fs.writeFileSync(htmlFile, html);
            fs.writeFileSync(xpathFile, xpath);
            
            // Run Go XPath
            const goPath = path.join(__dirname, 'go');
            const cmd = `cd "${goPath}" && go run main.go "${htmlFile}" "${xpathFile}"`;
            const output = execSync(cmd, { 
                encoding: 'utf8', 
                timeout: 10000,
                maxBuffer: 1024 * 1024
            });
            
            // Clean up temp files
            if (fs.existsSync(htmlFile)) fs.unlinkSync(htmlFile);
            if (fs.existsSync(xpathFile)) fs.unlinkSync(xpathFile);
            
            // Parse JSON output from Go
            const goResult = JSON.parse(output.trim());
            
            if (goResult.error) {
                throw new Error(goResult.error);
            }
            
            // Convert Go results to our format
            const nodes = [];
            if (goResult.results && Array.isArray(goResult.results)) {
                for (const result of goResult.results) {
                    nodes.push({
                        nodeName: result.nodeName ? result.nodeName.toLowerCase() : '',
                        textContent: result.textContent ? result.textContent.trim() : '',
                        attributes: result.attributes || {}
                    });
                }
            }
            
            return nodes;
        } catch (error) {
            throw new Error(`Go XPath error: ${error.message}`);
        }
    }

    getNodeAttributes(node) {
        const attrs = {};
        if (node.attributes) {
            for (let i = 0; i < node.attributes.length; i++) {
                const attr = node.attributes[i];
                attrs[attr.name] = attr.value;
            }
        }
        return attrs;
    }

    compareResults(jsResults, goResults) {
        if (jsResults.length !== goResults.length) {
            return false;
        }
        
        for (let i = 0; i < jsResults.length; i++) {
            const jsNode = jsResults[i];
            const goNode = goResults[i];
            
            if (jsNode.nodeName !== goNode.nodeName) {
                return false;
            }
            
            if (jsNode.textContent !== goNode.textContent) {
                return false;
            }
        }
        
        return true;
    }

    async runAllTests() {
        console.log('🚀 Starting Unified XPath Compatibility Test\n');
        
        const testCount = this.loadAllTestCases();
        if (testCount === 0) {
            console.log('❌ No test cases found');
            return;
        }

        let coreResults = { passed: 0, total: 0 };
        let extendedResults = { passed: 0, total: 0 };
        
        for (let i = 0; i < this.allTestCases.length; i++) {
            const testCase = this.allTestCases[i];
            
            process.stdout.write(`[${i + 1}/${this.allTestCases.length}] ${testCase.name}`);
            
            const result = await this.compareXPathResults(testCase);
            
            // Track by suite
            if (testCase.suite === 'core') {
                coreResults.total++;
                if (result.match) coreResults.passed++;
            } else {
                extendedResults.total++;
                if (result.match) extendedResults.passed++;
            }
            
            this.results.total++;
            if (result.match) {
                this.results.passed++;
                console.log(`\n✅ PASS - Results match perfectly`);
                console.log(`   Found ${result.jsCount} matching nodes`);
            } else {
                this.results.failed++;
                console.log(`\n❌ FAIL - Results don't match`);
                console.log(`   XPath: ${testCase.xpath}`);
                console.log(`   JavaScript: ${result.jsCount} results`);
                console.log(`   Go: ${result.goCount} results`);
                if (result.error) {
                    console.log(`   Error: ${result.error}`);
                }
            }
            
            this.results.details.push(result);
            console.log();
        }
        
        this.results.compatibility = ((this.results.passed / this.results.total) * 100);
        
        // Print detailed summary
        console.log('\n🎯 UNIFIED COMPATIBILITY REPORT');
        console.log('================================');
        console.log(`📊 CORE TESTS (Original): ${coreResults.passed}/${coreResults.total} (${((coreResults.passed/coreResults.total)*100).toFixed(1)}%)`);
        console.log(`📊 EXTENDED TESTS: ${extendedResults.passed}/${extendedResults.total} (${((extendedResults.passed/extendedResults.total)*100).toFixed(1)}%)`);
        console.log(`🎯 OVERALL: ${this.results.passed}/${this.results.total} (${this.results.compatibility.toFixed(1)}%)`);
        
        if (this.results.compatibility === 100) {
            console.log('\n🎉 PERFECT COMPATIBILITY ACHIEVED! 🎉');
        } else {
            console.log(`\n🔧 ${this.results.failed} tests need attention`);
        }
        
        // Save detailed report
        const reportPath = path.join(__dirname, 'unified_compatibility_report.json');
        fs.writeFileSync(reportPath, JSON.stringify(this.results, null, 2));
        console.log(`\n📊 Detailed report saved to: ${reportPath}`);
    }
}

// Run the tests
const tester = new UnifiedXPathTester();
tester.runAllTests().catch(console.error);