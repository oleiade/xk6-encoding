// This file contains a partial adaptation of the testharness.js implementation from
// the W3C Web Platform test suite. It is not intended to be a complete
// implementation, but rather a minimal set of functions to support the
// tests for this extension.
//
// Some of the function have been modified to support the k6 javascript runtime,
// and to limit its dependency to the rest of the W3C WebCrypto API test suite internal
// codebase.
//
// The original testharness.js implementation is available at:
// https://github.com/web-platform-tests/wpt/blob/3a3453c62176c97ab51cd492553c2dacd24366b1/resources/testharness.js

/**
 * Helper function for precise value comparison, handling NaN and signed zero.
 * Based on WPT testharness.js same_value function.
 *
 * @param {Any} x - First value to compare.
 * @param {Any} y - Second value to compare.
 * @returns {boolean} - True if values are the same.
 */
function same_value(x, y) {
  if (y !== y) {
    // NaN case - check if x is also NaN
    return x !== x;
  }
  if (x === 0 && y === 0) {
    // Distinguish +0 and -0
    return 1/x === 1/y;
  }
  return x === y;
}

/**
 * Assert that ``actual`` is the same value as ``expected``.
 *
 * For objects this compares by object identity; for primitives
 * this distinguishes between 0 and -0, and has correct handling
 * of NaN.
 *
 * @param {Any} actual - Test value.
 * @param {Any} expected - Expected value.
 * @param {string} [description] - Description of the condition being tested.
 */
function assert_equals(actual, expected, description) {
  // Check type equality first, as per WPT implementation
  if (typeof actual != typeof expected) {
    throw `assert_equals ${description} expected (${typeof expected}) ${expected} but got (${typeof actual}) ${actual}`;
  }
  
  // Use same_value for precise comparison
  if (!same_value(actual, expected)) {
    throw `assert_equals ${description} expected (${typeof expected}) ${expected} but got (${typeof actual}) ${actual}`;
  }
}

/**
 * Assert that ``actual`` is not the same value as ``expected``.
 *
 * Comparison is as for :js:func:`assert_equals`.
 *
 * @param {Any} actual - Test value.
 * @param {Any} expected - The value ``actual`` is expected to be different to.
 * @param {string} [description] - Description of the condition being tested.
 */
function assert_not_equals(actual, expected, description) {
  if (actual === expected) {
    throw `assert_not_equals ${description} got disallowed value ${actual}`;
  }
}

/**
 * Assert that ``actual`` is strictly true
 *
 * @param {Any} actual - Value that is asserted to be true
 * @param {string} [description] - Description of the condition being tested
 */
function assert_true(actual, description) {
  if (!actual) {
    throw `assert_true ${description} expected true got ${actual}`;
  }
}

/**
 * Assert that ``actual`` is strictly false
 *
 * @param {Any} actual - Value that is asserted to be false
 * @param {string} [description] - Description of the condition being tested
 */
function assert_false(actual, description) {
  if (actual) {
    throw `assert_true ${description} expected false got ${actual}`;
  }
}

/**
 * Assert that ``expected`` is an array and ``actual`` is one of the members.
 * This is implemented using ``indexOf``, so doesn't handle NaN or ±0 correctly.
 *
 * @param {Any} actual - Test value.
 * @param {Array} expected - An array that ``actual`` is expected to
 * be a member of.
 * @param {string} [description] - Description of the condition being tested.
 */
function assert_in_array(actual, expected, description) {
  if (expected.indexOf(actual) === -1) {
    throw `assert_in_array ${description} value ${actual} not in array ${expected}`;
  }
}

/**
 * Asserts if called. Used to ensure that a specific codepath is
 * not taken e.g. that an error event isn't fired.
 *
 * @param {string} [description] - Description of the condition being tested.
 */
function assert_unreached(description) {
  throw `reached unreachable code, reason: ${description}`;
}

/**
 * Assert a JS Error with the expected constructor is thrown.
 *
 * @param {object} constructor The expected exception constructor.
 * @param {Function} func Function which should throw.
 * @param {string} [description] Error description for the case that the error is not thrown.
 */
function assert_throws_js(constructor, func, description)
{
    try {
        func();
    } catch (e) {
        if (e instanceof constructor) {
        return;
        }
        throw `assert_throws_js ${description} expected ${constructor.name} but got ${e.name}`;
    }
    throw `assert_throws_js ${description} expected ${constructor.name} but no exception was thrown`;
}

/**
 * Run a test function with scope isolation.
 * This provides basic WPT compatibility by isolating test functions in their own scope.
 *
 * @param {Function} func The test function to run.
 * @param {string} [name] Optional test name for debugging.
 */
function test(func, name) {
    try {
        func();
    } catch (e) {
        if (name) {
            throw `Test "${name}" failed: ${e}`;
        }
        throw e;
    }
}

/**
 * Create a buffer for testing purposes.
 * This function creates ArrayBuffer instances for WPT compatibility.
 *
 * @param {string} _ The buffer type (ignored for compatibility).
 * @param {number} size The size of the buffer to create.
 * @returns {ArrayBuffer} A new ArrayBuffer of the specified size.
 */
function createBuffer(_, size) {
    return new ArrayBuffer(size);
}
