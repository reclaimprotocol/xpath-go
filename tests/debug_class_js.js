#!/usr/bin/env node

const { JSDOM } = require('jsdom');

function testClassLogic() {
    const html = `<html><body><div class='primary active large'>Item 1</div><div class='secondary active'>Item 2</div><div class='primary inactive'>Item 3</div></body></html>`;
    const xpath = `//div[contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))]`;
    
    console.log("=== TESTING COMPLEX CLASS LOGIC ===\n");
    console.log(`XPath: ${xpath}`);
    console.log("Expected: 1 result (Item 1 only)");
    
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
        
        console.log(`JavaScript found: ${result.snapshotLength} results`);
        
        for (let i = 0; i < result.snapshotLength; i++) {
            const node = result.snapshotItem(i);
            console.log(`  Result ${i + 1}: <${node.nodeName.toLowerCase()} class="${node.getAttribute('class')}">${node.textContent}</${node.nodeName.toLowerCase()}>`);
        }
        
        // Test each div individually to debug
        console.log("\n--- Testing each div individually ---");
        const allDivs = document.evaluate(
            "//div",
            document,
            null,
            dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
            null
        );
        
        for (let i = 0; i < allDivs.snapshotLength; i++) {
            const div = allDivs.snapshotItem(i);
            const classAttr = div.getAttribute('class');
            
            console.log(`\nDiv ${i+1}: class="${classAttr}" text="${div.textContent}"`);
            console.log(`  Raw class value: '${classAttr}'`);
            console.log(`  Length: ${classAttr ? classAttr.length : 'null'}`);
            console.log(`  contains(@class, 'primary'): ${classAttr && classAttr.includes('primary')}`);
            console.log(`  contains(@class, 'active'): ${classAttr && classAttr.includes('active')}`);
            console.log(`  contains(@class, 'inactive'): ${classAttr && classAttr.includes('inactive')}`);
            console.log(`  not(contains(@class, 'inactive')): ${!(classAttr && classAttr.includes('inactive'))}`);
            
            const hasPrimary = classAttr && classAttr.includes('primary');
            const hasActive = classAttr && classAttr.includes('active');
            const hasInactive = classAttr && classAttr.includes('inactive');
            const shouldMatch = hasPrimary && hasActive && !hasInactive;
            console.log(`  Should match: ${shouldMatch}`);
        }
        
    } catch (error) {
        console.error(`JavaScript XPath error: ${error.message}`);
    }
}

testClassLogic();