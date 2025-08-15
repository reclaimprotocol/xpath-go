#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const { JSDOM } = require('jsdom');

// JSDOM XPath compatibility tester
class XPathCompatibilityTester {
    constructor() {
        this.testCases = [];
        this.results = {
            total: 0,
            passed: 0,
            failed: 0,
            compatibility: 0,
            details: []
        };
    }

    loadTestCases() {
        const testCasesPath = path.join(__dirname, 'shared', 'testcases.json');
        if (fs.existsSync(testCasesPath)) {
            this.testCases = JSON.parse(fs.readFileSync(testCasesPath, 'utf8'));
        } else {
            // Default test cases for XPath
            this.testCases = [
                {
                    name: "Basic element selection",
                    html: "<html><body><div>Hello</div><p>World</p></body></html>",
                    xpath: "//div",
                    description: "Select all div elements"
                },
                {
                    name: "Attribute selection", 
                    html: "<html><body><div id='test' class='highlight'>Content</div></body></html>",
                    xpath: "//div[@id='test']",
                    description: "Select div with specific id"
                },
                {
                    name: "Text content selection",
                    html: "<html><body><p>First</p><p>Second</p><p>Third</p></body></html>",
                    xpath: "//p[text()='Second']",
                    description: "Select element by text content"
                },
                {
                    name: "Position-based selection",
                    html: "<html><body><ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul></body></html>",
                    xpath: "//li[position()=2]",
                    description: "Select second list item"
                },
                {
                    name: "Descendant axis",
                    html: "<html><body><div><span><a>Link</a></span></div></body></html>",
                    xpath: "//div//a",
                    description: "Select anchor descendant of div"
                }
            ];
        }
        console.log(`Loaded ${this.testCases.length} test cases`);
    }

    async runJavaScriptTest(html, xpath) {
        try {
            const dom = new JSDOM(html, {
                includeNodeLocations: true,
                features: {
                    FetchExternalResources: false,
                    ProcessExternalResources: false
                }
            });
            
            const document = dom.window.document;
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
                const location = dom.nodeLocation(node) || {};
                
                nodes.push({
                    value: node.nodeValue || node.textContent || '',
                    nodeName: node.nodeName.toLowerCase(),
                    nodeType: node.nodeType,
                    attributes: this.getAttributes(node),
                    textContent: node.textContent || '',
                    startLocation: location.startOffset || 0,
                    endLocation: location.endOffset || 0,
                    path: this.getNodePath(node)
                });
            }

            return {
                success: true,
                results: nodes,
                count: nodes.length,
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

    getAttributes(node) {
        const attrs = {};
        if (node.attributes) {
            for (let i = 0; i < node.attributes.length; i++) {
                const attr = node.attributes[i];
                attrs[attr.name] = attr.value;
            }
        }
        return attrs;
    }

    getNodePath(node) {
        const path = [];
        let current = node;
        
        while (current && current !== current.ownerDocument) {
            let selector = current.nodeName.toLowerCase();
            
            if (current.id) {
                selector += `[@id='${current.id}']`;
            } else if (current.className) {
                selector += `[@class='${current.className}']`;
            } else {
                // Add position if no unique identifier
                const siblings = Array.from(current.parentNode?.children || []);
                const index = siblings.indexOf(current) + 1;
                if (siblings.length > 1) {
                    selector += `[${index}]`;
                }
            }
            
            path.unshift(selector);
            current = current.parentNode;
        }
        
        return '/' + path.join('/');
    }

    async runGoTest(html, xpath) {
        try {
            // Write test data to temporary files
            const timestamp = Date.now();
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

            const result = JSON.parse(output.trim());
            return {
                success: true,
                results: result.results || [],
                count: result.count || 0,
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

    compareResults(jsResult, goResult) {
        if (!jsResult.success || !goResult.success) {
            return {
                match: false,
                reason: 'Execution error',
                jsError: jsResult.error,
                goError: goResult.error
            };
        }

        if (jsResult.count !== goResult.count) {
            return {
                match: false,
                reason: `Count mismatch: JS=${jsResult.count}, Go=${goResult.count}`,
                jsResults: jsResult.results,
                goResults: goResult.results
            };
        }

        // Compare each result
        for (let i = 0; i < jsResult.results.length; i++) {
            const jsNode = jsResult.results[i];
            const goNode = goResult.results[i];

            if (jsNode.nodeName !== goNode.nodeName) {
                return {
                    match: false,
                    reason: `Node name mismatch at index ${i}: JS="${jsNode.nodeName}", Go="${goNode.nodeName}"`,
                    jsResults: jsResult.results,
                    goResults: goResult.results
                };
            }

            if (jsNode.textContent !== goNode.textContent) {
                return {
                    match: false,
                    reason: `Text content mismatch at index ${i}: JS="${jsNode.textContent}", Go="${goNode.textContent}"`,
                    jsResults: jsResult.results,
                    goResults: goResult.results
                };
            }
        }

        return { match: true };
    }

    async runCompatibilityTest() {
        console.log('🧪 Starting XPath Compatibility Testing');
        console.log('=====================================');
        
        for (let i = 0; i < this.testCases.length; i++) {
            const testCase = this.testCases[i];
            this.results.total++;

            console.log(`\n[${i + 1}/${this.testCases.length}] ${testCase.name}`);
            console.log(`XPath: ${testCase.xpath}`);

            // Run JavaScript (jsdom) test
            const jsResult = await this.runJavaScriptTest(testCase.html, testCase.xpath);
            
            // Run Go test
            const goResult = await this.runGoTest(testCase.html, testCase.xpath);

            // Compare results
            const comparison = this.compareResults(jsResult, goResult);
            
            const testDetail = {
                name: testCase.name,
                xpath: testCase.xpath,
                passed: comparison.match,
                jsResult: jsResult,
                goResult: goResult,
                comparison: comparison
            };

            this.results.details.push(testDetail);

            if (comparison.match) {
                this.results.passed++;
                console.log('✅ PASS - Results match perfectly');
                console.log(`   Found ${jsResult.count} matching nodes`);
            } else {
                this.results.failed++;
                console.log('❌ FAIL - Results differ');
                console.log(`   Reason: ${comparison.reason}`);
                if (comparison.jsError) console.log(`   JS Error: ${comparison.jsError}`);
                if (comparison.goError) console.log(`   Go Error: ${comparison.goError}`);
            }
        }

        this.results.compatibility = ((this.results.passed / this.results.total) * 100).toFixed(1);
        this.generateReport();
    }

    generateReport() {
        console.log('\n🎯 COMPATIBILITY REPORT');
        console.log('========================');
        console.log(`Total Tests: ${this.results.total}`);
        console.log(`Passed: ${this.results.passed}`);
        console.log(`Failed: ${this.results.failed}`);
        console.log(`Compatibility: ${this.results.compatibility}%`);

        // Save detailed report
        const reportPath = path.join(__dirname, 'xpath_compatibility_report.json');
        fs.writeFileSync(reportPath, JSON.stringify(this.results, null, 2));
        console.log(`\n📊 Detailed report saved to: ${reportPath}`);

        if (this.results.compatibility === '100.0') {
            console.log('\n🎉 PERFECT COMPATIBILITY ACHIEVED! 🎉');
        } else {
            console.log(`\n🔧 ${this.results.failed} tests need attention for 100% compatibility`);
        }
    }
}

// Run if called directly
if (require.main === module) {
    const tester = new XPathCompatibilityTester();
    tester.loadTestCases();
    tester.runCompatibilityTest().catch(console.error);
}

module.exports = XPathCompatibilityTester;