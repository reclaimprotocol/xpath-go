#!/usr/bin/env node

const { JSDOM } = require('jsdom');

function testPosition() {
    // Test 1: Table navigation
    const html1 = `<html><body><table><thead><tr><th>Name</th><th>Age</th></tr></thead><tbody><tr><td>John</td><td>25</td></tr><tr><td>Jane</td><td>30</td></tr></tbody></table></body></html>`;
    const xpath1 = `//tbody/tr[position()>1]/td[position()=1]`;
    
    console.log("=== DEBUGGING POSITION() FUNCTION ===\n");
    console.log("Test 1: Table navigation");
    console.log(`XPath: ${xpath1}`);
    console.log("Expected: 1 result (Jane)");
    
    const dom = new JSDOM(html1);
    const document = dom.window.document;
    
    try {
        const result = document.evaluate(
            xpath1,
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        console.log(`JavaScript found: ${result.snapshotLength} results`);
        
        for (let i = 0; i < result.snapshotLength; i++) {
            const node = result.snapshotItem(i);
            console.log(`  Result ${i + 1}: <${node.nodeName.toLowerCase()}>${node.textContent}</${node.nodeName.toLowerCase()}>`);
        }
        
        // Test individual parts
        console.log("\n--- Breaking down the query ---");
        
        // First part: //tbody/tr[position()>1]
        const part1 = document.evaluate(
            "//tbody/tr[position()>1]",
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        console.log(`//tbody/tr[position()>1] found: ${part1.snapshotLength} results`);
        for (let i = 0; i < part1.snapshotLength; i++) {
            const node = part1.snapshotItem(i);
            console.log(`  TR ${i + 1}: ${node.textContent.trim()}`);
        }
        
        // Test all tr elements
        const allTRs = document.evaluate(
            "//tbody/tr",
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        console.log(`\nAll tbody/tr elements: ${allTRs.snapshotLength} results`);
        for (let i = 0; i < allTRs.snapshotLength; i++) {
            const node = allTRs.snapshotItem(i);
            console.log(`  TR ${i + 1} (position=${i+1}): ${node.textContent.trim()}`);
        }
        
    } catch (error) {
        console.error(`JavaScript XPath error: ${error.message}`);
    }
    
    console.log("\n");
    
    // Test 2: Position in filtered set
    const html2 = `<html><body><div><span class='item'>A</span><p>X</p><span class='item'>B</span><div>Y</div><span class='item'>C</span><span class='item'>D</span></div></body></html>`;
    const xpath2 = `//span[@class='item'][position() mod 2 = 0]`;
    
    console.log("Test 2: Position in filtered set");
    console.log(`XPath: ${xpath2}`);
    console.log("Expected: 2 results (B and D - even positions)");
    
    const dom2 = new JSDOM(html2);
    const document2 = dom2.window.document;
    
    try {
        const result2 = document2.evaluate(
            xpath2,
            document2,
            null,
            dom2.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        console.log(`JavaScript found: ${result2.snapshotLength} results`);
        
        for (let i = 0; i < result2.snapshotLength; i++) {
            const node = result2.snapshotItem(i);
            console.log(`  Result ${i + 1}: <${node.nodeName.toLowerCase()}>${node.textContent}</${node.nodeName.toLowerCase()}>`);
        }
        
        // Test the filtered set first
        console.log("\n--- Breaking down the query ---");
        const filtered = document2.evaluate(
            "//span[@class='item']",
            document2,
            null,
            dom2.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        console.log(`//span[@class='item'] found: ${filtered.snapshotLength} results`);
        for (let i = 0; i < filtered.snapshotLength; i++) {
            const node = filtered.snapshotItem(i);
            console.log(`  Span ${i + 1} (position=${i+1}): ${node.textContent}`);
            console.log(`    position() mod 2 = ${(i+1) % 2} (${(i+1) % 2 === 0 ? 'EVEN - should match' : 'ODD - should not match'})`);
        }
        
    } catch (error) {
        console.error(`JavaScript XPath error: ${error.message}`);
    }
}

testPosition();