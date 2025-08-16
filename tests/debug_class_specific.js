#!/usr/bin/env node

const { JSDOM } = require('jsdom');

function testSpecificClass() {
    const html = `<html><body><div class='primary inactive'>Item 3</div></body></html>`;
    
    console.log("=== TESTING SPECIFIC CLASS ISSUE ===\n");
    console.log("HTML: <div class='primary inactive'>Item 3</div>");
    
    const dom = new JSDOM(html);
    const document = dom.window.document;
    
    const div = document.querySelector('div');
    const classAttr = div.getAttribute('class');
    
    console.log(`\nClass attribute: "${classAttr}"`);
    console.log(`Length: ${classAttr.length}`);
    console.log(`Characters: ${JSON.stringify(classAttr.split(''))}`);
    
    // Test contains function manually
    console.log(`\nManual substring tests:`);
    console.log(`  classAttr.includes('active'): ${classAttr.includes('active')}`);
    console.log(`  classAttr.includes('inactive'): ${classAttr.includes('inactive')}`);
    console.log(`  classAttr.indexOf('active'): ${classAttr.indexOf('active')}`);
    console.log(`  classAttr.indexOf('inactive'): ${classAttr.indexOf('inactive')}`);
    
    // Test XPath contains
    console.log(`\nXPath contains tests:`);
    try {
        const containsActive = document.evaluate(
            "contains(@class, 'active')",
            div,
            null,
            dom.window.XPathResult.BOOLEAN_TYPE,
            null
        );
        console.log(`  XPath contains(@class, 'active'): ${containsActive.booleanValue}`);
        
        const containsInactive = document.evaluate(
            "contains(@class, 'inactive')",
            div,
            null,
            dom.window.XPathResult.BOOLEAN_TYPE,
            null
        );
        console.log(`  XPath contains(@class, 'inactive'): ${containsInactive.booleanValue}`);
        
    } catch (error) {
        console.error(`XPath error: ${error.message}`);
    }
}

testSpecificClass();