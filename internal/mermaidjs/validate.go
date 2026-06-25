package mermaidjs

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dop251/goja"
)

//go:embed mermaid.min.js
var mermaidJS string

// Browser/UMD environment stubs required for the mermaid bundle to initialize.
const jsPrefix = `
var self = this;
var window = this;
var globalThis = this;
var module = { exports: {} };
var exports = module.exports;

// window-level event stubs (mermaid may call window.addEventListener)
this.addEventListener = function() {};
this.removeEventListener = function() {};
this.dispatchEvent = function() { return true; };

var document = (function() {
  function el(tag) {
    return {
      tagName: tag || 'div', style: {}, className: '', innerHTML: '', textContent: '',
      getAttribute: function() { return null; },
      setAttribute: function() {},
      removeAttribute: function() {},
      hasAttribute: function() { return false; },
      appendChild: function(c) { return c || {}; },
      removeChild: function() {},
      insertBefore: function(c) { return c || {}; },
      querySelector: function() { return null; },
      querySelectorAll: function() { return []; },
      getElementsByTagName: function() { return []; },
      getBoundingClientRect: function() { return {width:0,height:0,top:0,left:0,right:0,bottom:0}; },
      addEventListener: function() {},
      removeEventListener: function() {},
      cloneNode: function() { return el(tag); },
      contains: function() { return false; },
      dispatchEvent: function() { return true; },
      closest: function() { return null; },
      matches: function() { return false; },
      nodeType: 1,
      childNodes: [],
      children: []
    };
  }
  var body = el('body');
  var head = el('head');
  return {
    createElement: el,
    createElementNS: function(ns, tag) { return el(tag); },
    createTextNode: function(t) { return { textContent: t, nodeType: 3, data: t }; },
    createComment: function() { return { nodeType: 8 }; },
    createDocumentFragment: function() { return el('fragment'); },
    querySelector: function() { return null; },
    querySelectorAll: function() { return []; },
    getElementById: function() { return null; },
    getElementsByTagName: function() { return []; },
    getElementsByClassName: function() { return []; },
    body: body, head: head,
    documentElement: el('html'),
    defaultView: null,
    readyState: 'complete',
    nodeType: 9,
    addEventListener: function() {},
    removeEventListener: function() {},
    createRange: function() {
      return {
        setStart: function() {}, setEnd: function() {},
        getBoundingClientRect: function() { return {width:0,height:0,top:0,left:0}; },
        getClientRects: function() { return []; },
        commonAncestorContainer: el('div')
      };
    }
  };
})();

var navigator = { userAgent: 'Mozilla/5.0 (ariel)', platform: '', language: 'en', languages: ['en'] };
var location = { href: 'http://localhost/', origin: 'http://localhost', protocol: 'http:', host: 'localhost', pathname: '/', search: '', hash: '' };
var history = { pushState: function() {}, replaceState: function() {}, state: null };
var performance = { now: function() { return Date.now(); }, mark: function() {}, measure: function() {}, getEntriesByName: function() { return []; } };
var screen = { width: 1920, height: 1080 };

var Event = function(type, init) { this.type = type; this.bubbles = (init && init.bubbles) || false; };
var CustomEvent = function(type, init) { this.type = type; this.detail = init && init.detail; };
var MouseEvent = function(type) { this.type = type; };
var KeyboardEvent = function(type) { this.type = type; };

var MutationObserver = function() { return { observe: function() {}, disconnect: function() {}, takeRecords: function() { return []; } }; };
var ResizeObserver = function() { return { observe: function() {}, disconnect: function() {} }; };
var IntersectionObserver = function() { return { observe: function() {}, disconnect: function() {} }; };

var requestAnimationFrame = function() { return 0; };
var cancelAnimationFrame = function() {};
var requestIdleCallback = function() { return 0; };
var cancelIdleCallback = function() {};

var getComputedStyle = function() { return { getPropertyValue: function() { return ''; }, setProperty: function() {} }; };
var matchMedia = function() { return { matches: false, addListener: function() {}, removeListener: function() {}, addEventListener: function() {}, removeEventListener: function() {} }; };

var SVGElement = function() {};
var HTMLElement = function() {};
var Element = function() {};

var crypto = { getRandomValues: function(a) { return a; }, randomUUID: function() { return '00000000-0000-4000-8000-000000000000'; } };
var XMLHttpRequest = function() { this.open = function() {}; this.send = function() {}; this.setRequestHeader = function() {}; this.addEventListener = function() {}; };
var fetch = function() { return Promise.resolve({ json: function() { return Promise.resolve({}); }, text: function() { return Promise.resolve(''); } }); };
var localStorage = { getItem: function() { return null; }, setItem: function() {}, removeItem: function() {} };
var sessionStorage = { getItem: function() { return null; }, setItem: function() {}, removeItem: function() {} };
var URL = function(u) { this.href = u; this.toString = function() { return u; }; };
var DOMPurify = { sanitize: function(s) { return s; }, isSupported: true };
this.DOMPurify = DOMPurify;
var structuredClone = function(obj) { return JSON.parse(JSON.stringify(obj)); };
var console = { log: function() {}, warn: function() {}, error: function() {}, info: function() {}, debug: function() {}, trace: function() {}, group: function() {}, groupEnd: function() {}, time: function() {}, timeEnd: function() {} };
`

const jsSuffix = `
var _mermaid = module.exports;
_mermaid.initialize({ startOnLoad: false, securityLevel: 'loose' });

var _validateResult = null;

// startValidate begins an async mermaid parse. Call getValidateResult() after
// this returns to retrieve the outcome (goja drains microtasks between calls).
function startValidate(diagramStr) {
  _validateResult = null;
  try {
    _mermaid.parse(diagramStr)
      .then(function() { _validateResult = JSON.stringify({ valid: true, error: null }); })
      .catch(function(e) { _validateResult = JSON.stringify({ valid: false, error: e.str || e.message || String(e) }); });
  } catch(e) {
    _validateResult = JSON.stringify({ valid: false, error: e.message || String(e) });
  }
}

function getValidateResult() {
  return _validateResult;
}
`

var (
	compiled     *goja.Program
	compileOnce  sync.Once
	compileErr   error
)

// getCompiled compiles the combined Mermaid+shim JS once and caches the result;
// subsequent calls return the cached program without recompiling.
func getCompiled() (*goja.Program, error) {
	compileOnce.Do(func() {
		combined := jsPrefix + mermaidJS + jsSuffix
		compiled, compileErr = goja.Compile("mermaid-validator", combined, false)
	})
	return compiled, compileErr
}

type validationResult struct {
	Valid bool    `json:"valid"`
	Error *string `json:"error"`
}

// Validate returns nil if diagram is valid Mermaid syntax, or an error with the
// parser's own message. Runs a fresh goja VM per call; each call is independent.
func Validate(diagram string) error {
	prog, err := getCompiled()
	if err != nil {
		return fmt.Errorf("mermaid validator compile: %w", err)
	}

	vm := goja.New()
	if _, err := vm.RunProgram(prog); err != nil {
		return fmt.Errorf("mermaid validator init: %w", err)
	}

	startFn, ok := goja.AssertFunction(vm.Get("startValidate"))
	if !ok {
		return fmt.Errorf("mermaid validator: startValidate not found")
	}
	if _, err := startFn(goja.Undefined(), vm.ToValue(diagram)); err != nil {
		return fmt.Errorf("mermaid validator call: %w", err)
	}

	// After startFn returns, goja drains the microtask queue (Promise callbacks run).
	// getValidateResult reads the result set by those callbacks.
	getFn, ok := goja.AssertFunction(vm.Get("getValidateResult"))
	if !ok {
		return fmt.Errorf("mermaid validator: getValidateResult not found")
	}
	res, err := getFn(goja.Undefined())
	if err != nil {
		return fmt.Errorf("mermaid validator get result: %w", err)
	}

	if res.Export() == nil {
		return fmt.Errorf("mermaid validator: result not set (promise did not resolve synchronously)")
	}

	var result validationResult
	if err := json.Unmarshal([]byte(res.String()), &result); err != nil {
		return fmt.Errorf("mermaid validator: failed to parse result: %w", err)
	}

	if !result.Valid {
		if result.Error != nil && *result.Error != "" {
			return fmt.Errorf("%s", *result.Error)
		}
		return fmt.Errorf("invalid mermaid syntax")
	}
	return nil
}
