#!/usr/bin/env node

const { JSDOM } = require('jsdom');

function testNegativeSubstring() {
    const html = `<html><body><p>Mid</p></body></html>`;
    
    console.log("=== TESTING VARIOUS SUBSTRING POSITIONS ===\n");
    
    const dom = new JSDOM(html);
    const document = dom.window.document;
    
    const positions = [-1, 0, 1, 2, 3, 4];
    
    for (const pos of positions) {
        try {
            const xpath = `//p[substring(text(), ${pos}) = "Mid"]`;
            const result = document.evaluate(
                xpath,
                document,
                null,
                dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
                null
            );
            
            console.log(`Position ${pos}: ${result.snapshotLength} results (substring(text(), ${pos}) = "Mid")`);
            
            // Also test what the actual substring returns by checking different values
            const tests = ["", "Mid", "id", "d"];
            for (const testVal of tests) {
                const testXpath = `//p[substring(text(), ${pos}) = "${testVal}"]`;
                const testResult = document.evaluate(
                    testXpath,
                    document,
                    null,
                    dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
                    null
                );
                if (testResult.snapshotLength > 0) {
                    console.log(`  -> substring(text(), ${pos}) equals "${testVal}"`);
                }
            }
        } catch (error) {
            console.log(`Position ${pos}: ERROR - ${error.message}`);
        }
    }
}

testNegativeSubstring();