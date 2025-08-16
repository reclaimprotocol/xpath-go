#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const { JSDOM } = require('jsdom');

// Comprehensive XPath compatibility tester
class ComprehensiveXPathTester {
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
        // Load original test cases
        const originalPath = path.join(__dirname, 'shared', 'testcases.json');
        if (fs.existsSync(originalPath)) {
            const original = JSON.parse(fs.readFileSync(originalPath, 'utf8'));
            this.allTestCases = this.allTestCases.concat(original.map(tc => ({...tc, suite: 'original'})));
        }

        // Load extended test cases
        const extendedPath = path.join(__dirname, 'shared', 'extended_testcases.json');
        if (fs.existsSync(extendedPath)) {
            const extended = JSON.parse(fs.readFileSync(extendedPath, 'utf8'));
            this.allTestCases = this.allTestCases.concat(extended.map(tc => ({...tc, suite: 'extended'})));
        }

        console.log(`\n📚 Loaded ${this.allTestCases.length} test cases:`);
        console.log(`   • Original suite: ${this.allTestCases.filter(tc => tc.suite === 'original').length} tests`);
        console.log(`   • Extended suite: ${this.allTestCases.filter(tc => tc.suite === 'extended').length} tests`);
        console.log();
    }

    async runComprehensiveTests() {
        console.log('🧪 Starting Comprehensive XPath Compatibility Testing');
        console.log('=====================================================\\n');

        this.loadAllTestCases();
        this.results.total = this.allTestCases.length;

        for (let i = 0; i < this.allTestCases.length; i++) {
            const testCase = this.allTestCases[i];
            const testNumber = i + 1;
            
            console.log(`[${testNumber}/${this.allTestCases.length}] ${testCase.name} ${testCase.suite === 'extended' ? '🆕' : ''}`);
            console.log(`XPath: ${testCase.xpath}`);

            const result = await this.runSingleTest(testCase);
            this.results.details.push(result);

            if (result.passed) {
                this.results.passed++;
                console.log(`✅ PASS - Results match perfectly`);
                console.log(`   Found ${result.jsResult.count} matching nodes\\n`);
            } else {
                this.results.failed++;
                console.log(`❌ FAIL - Results differ`);
                console.log(`   Reason: ${result.comparison.reason}\\n`);
            }
        }

        this.generateReport();
    }

    async runSingleTest(testCase) {
        const jsResult = this.runJavaScriptTest(testCase.xpath, testCase.html);
        const goResult = this.runGoTest(testCase.xpath, testCase.html);
        
        const comparison = this.compareResults(jsResult, goResult);

        return {
            name: testCase.name,
            xpath: testCase.xpath,
            suite: testCase.suite,
            category: testCase.category,
            passed: comparison.match,
            jsResult,
            goResult,
            comparison
        };
    }

    runJavaScriptTest(xpath, html) {
        try {
            const dom = new JSDOM(html);
            const document = dom.window.document;
            const window = dom.window;
            
            const xpathResult = document.evaluate(
                xpath,
                document,
                null,
                window.XPathResult.ANY_TYPE,
                null
            );

            const results = [];
            let node;

            // Handle different result types
            switch (xpathResult.resultType) {
                case window.XPathResult.ORDERED_NODE_ITERATOR_TYPE:
                case window.XPathResult.UNORDERED_NODE_ITERATOR_TYPE:
                    while (node = xpathResult.iterateNext()) {
                        results.push(this.nodeToResult(node));
                    }
                    break;
                    
                case window.XPathResult.FIRST_ORDERED_NODE_TYPE:
                    if (xpathResult.singleNodeValue) {
                        results.push(this.nodeToResult(xpathResult.singleNodeValue));
                    }
                    break;
                    
                case window.XPathResult.STRING_TYPE:
                    results.push({
                        value: xpathResult.stringValue,
                        nodeName: "#text",
                        nodeType: 3,
                        attributes: {},
                        textContent: xpathResult.stringValue,
                        startLocation: 0,
                        endLocation: 0,
                        path: "/string-result"
                    });
                    break;
            }

            return {
                success: true,
                results: results,
                count: results.length,
                error: null
            };

        } catch (error) {
            return {
                success: false,
                results: [],
                count: 0,
                error: error.message
            };
        }
    }

    runGoTest(xpath, html) {
        try {
            // Create temporary files
            const timestamp = Date.now() + '_' + Math.random().toString(36).substr(2, 9);
            const htmlFile = path.join(__dirname, `temp_html_${timestamp}.html`);
            const xpathFile = path.join(__dirname, `temp_xpath_${timestamp}.txt`);

            fs.writeFileSync(htmlFile, html);
            fs.writeFileSync(xpathFile, xpath);

            // Run Go test program
            const goTestPath = path.join(__dirname, 'go', 'main.go');
            const cmd = `cd "${path.dirname(goTestPath)}" && go run main.go "${htmlFile}" "${xpathFile}"`;
            
            const output = execSync(cmd, { 
                encoding: 'utf8',
                timeout: 10000,
                maxBuffer: 1024 * 1024
            });

            // Clean up temporary files
            fs.unlinkSync(htmlFile);
            fs.unlinkSync(xpathFile);

            const goResult = JSON.parse(output.trim());
            
            if (goResult.error) {
                return {
                    success: false,
                    results: [],
                    count: 0,
                    error: goResult.error
                };
            }

            return {
                success: true,
                results: goResult.results || [],
                count: goResult.count || 0,
                error: null
            };

        } catch (error) {
            return {
                success: false,
                results: [],
                count: 0,
                error: error.message
            };
        }
    }

    nodeToResult(node) {
        // Calculate approximate positions
        const startLocation = this.getNodePosition(node);
        
        return {
            value: node.nodeValue || node.textContent || "",
            nodeName: node.nodeName ? node.nodeName.toLowerCase() : node.name,
            nodeType: node.nodeType,
            attributes: this.getNodeAttributes(node),
            textContent: node.textContent || "",
            startLocation: startLocation,
            endLocation: startLocation,
            path: this.getNodePath(node)
        };
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

    getNodePosition(node) {
        // Simple position calculation - could be enhanced
        let pos = 0;
        let current = node;
        while (current && current.parentNode) {
            let sibling = current.parentNode.firstChild;
            while (sibling && sibling !== current) {
                pos += (sibling.textContent || sibling.outerHTML || "").length;
                sibling = sibling.nextSibling;
            }
            current = current.parentNode;
        }
        return pos;
    }

    getNodePath(node) {
        const path = [];
        let current = node;
        
        while (current && current.nodeType !== 9) { // Not document node
            let name = current.nodeName.toLowerCase();
            let position = 1;
            
            if (current.previousSibling) {
                let sibling = current.previousSibling;
                while (sibling) {
                    if (sibling.nodeName === current.nodeName) {
                        position++;
                    }
                    sibling = sibling.previousSibling;
                }
            }
            
            if (position > 1 || this.hasNextSiblingWithSameName(current)) {
                name += `[${position}]`;
            }
            
            path.unshift(name);
            current = current.parentNode;
        }
        
        return '/' + path.join('/');
    }

    hasNextSiblingWithSameName(node) {
        let sibling = node.nextSibling;
        while (sibling) {
            if (sibling.nodeName === node.nodeName) {
                return true;
            }
            sibling = sibling.nextSibling;
        }
        return false;
    }

    compareResults(jsResult, goResult) {
        // If both have errors, compare error messages
        if (!jsResult.success && !goResult.success) {
            return {
                match: jsResult.error === goResult.error,
                reason: jsResult.error === goResult.error ? "Both failed with same error" : "Different errors",
                jsResults: [],
                goResults: []
            };
        }

        // If one succeeded and one failed
        if (jsResult.success !== goResult.success) {
            return {
                match: false,
                reason: `Success mismatch: JS=${jsResult.success}, Go=${goResult.success}`,
                jsResults: jsResult.results,
                goResults: goResult.results
            };
        }

        // Both succeeded, compare results
        if (jsResult.count !== goResult.count) {
            return {
                match: false,
                reason: `Count mismatch: JS=${jsResult.count}, Go=${goResult.count}`,
                jsResults: jsResult.results,
                goResults: goResult.results
            };
        }

        // Compare individual results
        for (let i = 0; i < jsResult.results.length; i++) {
            const jsRes = jsResult.results[i];
            const goRes = goResult.results[i];

            // Compare node names
            if (jsRes.nodeName !== goRes.nodeName) {
                return {
                    match: false,
                    reason: `Node name mismatch at index ${i}: JS="${jsRes.nodeName}", Go="${goRes.nodeName}"`,
                    jsResults: jsResult.results,
                    goResults: goResult.results
                };
            }

            // Compare text content
            if (jsRes.textContent !== goRes.textContent) {
                return {
                    match: false,
                    reason: `Text content mismatch at index ${i}: JS="${jsRes.textContent}", Go="${goRes.textContent}"`,
                    jsResults: jsResult.results,
                    goResults: goResult.results
                };
            }

            // Compare node types
            if (jsRes.nodeType !== goRes.nodeType) {
                return {
                    match: false,
                    reason: `Node type mismatch at index ${i}: JS=${jsRes.nodeType}, Go=${goRes.nodeType}`,
                    jsResults: jsResult.results,
                    goResults: goResult.results
                };
            }
        }

        return {
            match: true,
            reason: "Perfect match",
            jsResults: jsResult.results,
            goResults: goResult.results
        };
    }

    generateReport() {
        this.results.compatibility = (this.results.passed / this.results.total * 100).toFixed(1);

        console.log('🎯 COMPREHENSIVE COMPATIBILITY REPORT');
        console.log('=====================================');
        console.log(`Total Tests: ${this.results.total}`);
        console.log(`Passed: ${this.results.passed}`);
        console.log(`Failed: ${this.results.failed}`);
        console.log(`Overall Compatibility: ${this.results.compatibility}%\\n`);

        // Breakdown by suite
        const originalResults = this.results.details.filter(r => r.suite === 'original');
        const extendedResults = this.results.details.filter(r => r.suite === 'extended');
        
        const originalPassed = originalResults.filter(r => r.passed).length;
        const extendedPassed = extendedResults.filter(r => r.passed).length;

        console.log('📊 DETAILED BREAKDOWN');
        console.log('=====================');
        console.log(`Original Suite: ${originalPassed}/${originalResults.length} (${(originalPassed/originalResults.length*100).toFixed(1)}%)`);
        console.log(`Extended Suite: ${extendedPassed}/${extendedResults.length} (${(extendedPassed/extendedResults.length*100).toFixed(1)}%)`);

        // Category breakdown for extended tests
        if (extendedResults.length > 0) {
            console.log('\\n🏷️ CATEGORY BREAKDOWN (Extended Tests)');
            console.log('======================================');
            const categories = {};
            extendedResults.forEach(result => {
                const cat = result.category || 'uncategorized';
                if (!categories[cat]) {
                    categories[cat] = { total: 0, passed: 0 };
                }
                categories[cat].total++;
                if (result.passed) categories[cat].passed++;
            });

            Object.keys(categories).sort().forEach(cat => {
                const { total, passed } = categories[cat];
                const percentage = (passed / total * 100).toFixed(1);
                console.log(`${cat.padEnd(20)}: ${passed}/${total} (${percentage}%)`);
            });
        }

        if (this.results.failed > 0) {
            console.log('\\n❌ FAILING TESTS');
            console.log('=================');
            this.results.details.filter(r => !r.passed).forEach((result, index) => {
                console.log(`${index + 1}. ${result.name} ${result.suite === 'extended' ? '🆕' : ''}`);
                console.log(`   XPath: ${result.xpath}`);
                console.log(`   Reason: ${result.comparison.reason}`);
                console.log();
            });

            console.log(`🔧 ${this.results.failed} tests need attention for 100% compatibility`);
        } else {
            console.log('\\n🎉 PERFECT COMPATIBILITY ACHIEVED! 🎉');
        }

        // Save detailed report
        const reportPath = path.join(__dirname, 'comprehensive_compatibility_report.json');
        fs.writeFileSync(reportPath, JSON.stringify(this.results, null, 2));
        console.log(`\\n📊 Detailed report saved to: ${reportPath}`);
    }
}

// Run the comprehensive tests
const tester = new ComprehensiveXPathTester();
tester.runComprehensiveTests().catch(error => {
    console.error('Test runner error:', error);
    process.exit(1);
});