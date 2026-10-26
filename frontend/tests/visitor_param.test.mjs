import test from "node:test";
import assert from "node:assert/strict";

import {
  parseVisitorParam,
  stripVisitorFromUrl,
  stripVisitorParam,
} from "../src/lib/visitor-param.mjs";

/** `v` + n lowercase alphanumerics. */
const id = (n) => "v" + "a1".repeat(Math.ceil(n / 2)).slice(0, n);

test("parseVisitorParam: length bounds of the id after `v`", () => {
  assert.equal(parseVisitorParam(`?spark_vid=${id(5)}`), "", "5 chars after v is too short");
  assert.equal(parseVisitorParam(`?spark_vid=${id(6)}`), id(6), "6 chars after v is valid");
  assert.equal(parseVisitorParam(`?spark_vid=${id(40)}`), id(40), "40 chars after v is valid");
  assert.equal(parseVisitorParam(`?spark_vid=${id(41)}`), "", "41 chars after v is too long");
});

test("parseVisitorParam: id must be lowercase alphanumerics with a v prefix", () => {
  assert.equal(parseVisitorParam("?spark_vid=vtEST123abcde"), "", "uppercase is rejected");
  assert.equal(parseVisitorParam("?spark_vid=vTEST123ABCDE"), "", "all-uppercase is rejected");
  assert.equal(parseVisitorParam("?spark_vid=vtest-1234567"), "", "dashes are rejected");
  assert.equal(parseVisitorParam("?spark_vid=test12345678"), "", "missing v prefix");
  assert.equal(parseVisitorParam("?spark_vid=bda11f7-90fa-4698-a739-12d7265ebbed"), "", "uuid is rejected");
  assert.equal(parseVisitorParam("?spark_vid=vtest_1234567"), "", "underscore is rejected");
});

test("parseVisitorParam: absent, empty and non-spark parameters", () => {
  assert.equal(parseVisitorParam(""), "");
  assert.equal(parseVisitorParam("?"), "");
  assert.equal(parseVisitorParam("?utm_source=youtube"), "");
  assert.equal(parseVisitorParam("?spark_vid="), "", "empty value is rejected");
  assert.equal(parseVisitorParam("?spark_vid"), "", "missing value is rejected");
  assert.equal(parseVisitorParam("?spark_visitor_id=vtest123456"), "", "other keys do not match");
});

test("parseVisitorParam: reads with or without a leading `?` and decodes the key", () => {
  assert.equal(parseVisitorParam(`spark_vid=${id(12)}`), id(12));
  assert.equal(parseVisitorParam(`?utm_source=x&spark_vid=${id(12)}&utm_medium=y`), id(12));
  assert.equal(parseVisitorParam(`?spark%5Fvid=${id(12)}`), id(12));
  assert.equal(parseVisitorParam("?spark_vid=vtest%20123456"), "", "%20 decodes to a space");
  assert.equal(parseVisitorParam("?spark_vid=vtest+1234567"), "", "+ decodes to a space");
});

test("stripVisitorParam: removes the parameter and keeps the rest verbatim", () => {
  assert.equal(
    stripVisitorParam("?spark_vid=vvalid123456&utm_source=youtube&utm_campaign=spring"),
    "?utm_source=youtube&utm_campaign=spring",
  );
  assert.equal(stripVisitorParam("?utm_source=youtube&spark_vid=vvalid123456"), "?utm_source=youtube");
  assert.equal(stripVisitorParam("?spark_vid=vvalid123456"), "");
  assert.equal(stripVisitorParam(""), "");
  assert.equal(stripVisitorParam("?"), "");
  assert.equal(stripVisitorParam("?utm_source=x"), "?utm_source=x", "no-op keeps the string");
});

test("stripVisitorParam: keeps %20 and + encoding untouched", () => {
  assert.equal(
    stripVisitorParam("?utm_content=a%20b&spark_vid=vvalid123456&q=c+d&r=%D0%9F"),
    "?utm_content=a%20b&q=c+d&r=%D0%9F",
  );
});

test("stripVisitorParam: keeps duplicate parameters, drops duplicate spark_vid", () => {
  assert.equal(
    stripVisitorParam("?utm_source=a&utm_source=b&spark_vid=vvalid123456"),
    "?utm_source=a&utm_source=b",
  );
  assert.equal(
    stripVisitorParam("?spark_vid=vvalid123456&spark_vid=vother123456&utm_source=a"),
    "?utm_source=a",
  );
});

test("stripVisitorFromUrl: path and hash are untouched", () => {
  assert.equal(
    stripVisitorFromUrl("/?spark_vid=vvalid123456&utm_source=youtube#pricing"),
    "/?utm_source=youtube#pricing",
  );
  assert.equal(stripVisitorFromUrl("/?spark_vid=vvalid123456#pricing"), "/#pricing");
  assert.equal(stripVisitorFromUrl("/privacy#top"), "/privacy#top", "no-op keeps the url");
  assert.equal(stripVisitorFromUrl("/"), "/");
  assert.equal(
    stripVisitorFromUrl("/offer?spark_vid=vvalid123456&plan=a%20b#cta"),
    "/offer?plan=a%20b#cta",
  );
});
