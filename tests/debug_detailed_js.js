#!/usr/bin/env node

const { JSDOM } = require('jsdom');

function testDetailedSubstring() {
    const html = `<html><body><p>Mid</p></body></html>`;
    
    console.log("=== DETAILED SUBSTRING TESTING ===\n");
    
    const dom = new JSDOM(html);
    const document = dom.window.document;
    
    // Test substring(text(), 0) with various lengths
    const lengths = [1, 2, 3, 4, 5];
    
    for (const len of lengths) {
        try {
            // Test what value actually matches
            const chars = ["", "M", "i", "d", "Mi", "id", "Mid", "Midd"];
            
            for (const testChar of chars) {
                const xpath = `//p[substring(text(), 0, ${len}) = "${testChar}"]`;
                const result = document.evaluate(
                    xpath,
                    document,
                    null,
                    dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
                    null
                );
                
                if (result.snapshotLength > 0) {
                    console.log(`substring("Mid", 0, ${len}) = "${testChar}"`);
                }
            }
        } catch (error) {
            console.log(`Length ${len}: ERROR - ${error.message}`);
        }
    }
    
    // Also test the no-length case more systematically
    console.log("\n--- Testing substring(text(), 0) without length ---");
    const chars = ["", "M", "i", "d", "Mi", "id", "Mid", "Midd"];
    
    for (const testChar of chars) {
        try {
            const xpath = `//p[substring(text(), 0) = "${testChar}"]`;
            const result = document.evaluate(
                xpath,
                document,
                null,
                dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
                null
            );
            
            if (result.snapshotLength > 0) {
                console.log(`substring("Mid", 0) = "${testChar}"`);
            }
        } catch (error) {
            console.log(`Testing "${testChar}": ERROR - ${error.message}`);
        }
    }
}

testDetailedSubstring();