const assert = require('assert');
const fs = require('fs');
const path = require('path');
const { JSDOM } = require('jsdom');

const cases = JSON.parse(fs.readFileSync(
    path.join(__dirname, 'shared', 'typed_conversion_testcases.json'),
    'utf8'
));

for (const testCase of cases) {
    const dom = new JSDOM(testCase.html);
    const document = dom.window.document;

    const result = document.evaluate(
        testCase.xpath,
        document,
        null,
        dom.window.XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
        null
    );
    const got = [];
    for (let i = 0; i < result.snapshotLength; i++) {
        got.push(result.snapshotItem(i).id);
    }
    assert.deepStrictEqual(got, testCase.want_ids, testCase.name);
}

console.log(`typed conversion browser oracle: ${cases.length} cases passed`);
