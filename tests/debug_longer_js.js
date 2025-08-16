#!/usr/bin/env node

const { JSDOM } = require('jsdom');

function testLongerSubstring() {
    const html = `<html><body><p>Hello</p></body></html>`;
    
    console.log("=== TESTING WITH LONGER STRING ===");
    console.log('Testing "Hello" with various positions:\n');
    
    const dom = new JSDOM(html);
    const document = dom.window.document;
    
    const positions = [-2, -1, 0, 1, 2, 3, 4, 5, 6, 7];
    
    for (const pos of positions) {
        try {
            // Test what each position returns
            const tests = ["", "H", "e", "l", "o", "He", "el", "ll", "lo", "Hello", "ello", "llo", "lo", "o"];
            
            for (const testVal of tests) {
                const xpath = `//p[substring(text(), ${pos}) = "${testVal}"]`;
                const result = document.evaluate(
                    xpath,
                    document,
                    null,
                    dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
                    null
                );
                
                if (result.snapshotLength > 0) {
                    console.log(`substring("Hello", ${pos}) = "${testVal}"`);
                    break; // Only show the first match for each position
                }
            }
        } catch (error) {
            console.log(`Position ${pos}: ERROR - ${error.message}`);
        }
    }
}

testLongerSubstring();