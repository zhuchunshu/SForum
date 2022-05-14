/******/ (() => { // webpackBootstrap
/******/ 	var __webpack_modules__ = ({

/***/ "./node_modules/@babel/runtime/regenerator/index.js":
/*!**********************************************************!*\
  !*** ./node_modules/@babel/runtime/regenerator/index.js ***!
  \**********************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

module.exports = __webpack_require__(/*! regenerator-runtime */ "./node_modules/regenerator-runtime/runtime.js");


/***/ }),

/***/ "./node_modules/axios/index.js":
/*!*************************************!*\
  !*** ./node_modules/axios/index.js ***!
  \*************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

module.exports = __webpack_require__(/*! ./lib/axios */ "./node_modules/axios/lib/axios.js");

/***/ }),

/***/ "./node_modules/axios/lib/adapters/xhr.js":
/*!************************************************!*\
  !*** ./node_modules/axios/lib/adapters/xhr.js ***!
  \************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./../utils */ "./node_modules/axios/lib/utils.js");
var settle = __webpack_require__(/*! ./../core/settle */ "./node_modules/axios/lib/core/settle.js");
var cookies = __webpack_require__(/*! ./../helpers/cookies */ "./node_modules/axios/lib/helpers/cookies.js");
var buildURL = __webpack_require__(/*! ./../helpers/buildURL */ "./node_modules/axios/lib/helpers/buildURL.js");
var buildFullPath = __webpack_require__(/*! ../core/buildFullPath */ "./node_modules/axios/lib/core/buildFullPath.js");
var parseHeaders = __webpack_require__(/*! ./../helpers/parseHeaders */ "./node_modules/axios/lib/helpers/parseHeaders.js");
var isURLSameOrigin = __webpack_require__(/*! ./../helpers/isURLSameOrigin */ "./node_modules/axios/lib/helpers/isURLSameOrigin.js");
var createError = __webpack_require__(/*! ../core/createError */ "./node_modules/axios/lib/core/createError.js");

module.exports = function xhrAdapter(config) {
  return new Promise(function dispatchXhrRequest(resolve, reject) {
    var requestData = config.data;
    var requestHeaders = config.headers;

    if (utils.isFormData(requestData)) {
      delete requestHeaders['Content-Type']; // Let the browser set it
    }

    var request = new XMLHttpRequest();

    // HTTP basic authentication
    if (config.auth) {
      var username = config.auth.username || '';
      var password = config.auth.password ? unescape(encodeURIComponent(config.auth.password)) : '';
      requestHeaders.Authorization = 'Basic ' + btoa(username + ':' + password);
    }

    var fullPath = buildFullPath(config.baseURL, config.url);
    request.open(config.method.toUpperCase(), buildURL(fullPath, config.params, config.paramsSerializer), true);

    // Set the request timeout in MS
    request.timeout = config.timeout;

    // Listen for ready state
    request.onreadystatechange = function handleLoad() {
      if (!request || request.readyState !== 4) {
        return;
      }

      // The request errored out and we didn't get a response, this will be
      // handled by onerror instead
      // With one exception: request that using file: protocol, most browsers
      // will return status as 0 even though it's a successful request
      if (request.status === 0 && !(request.responseURL && request.responseURL.indexOf('file:') === 0)) {
        return;
      }

      // Prepare the response
      var responseHeaders = 'getAllResponseHeaders' in request ? parseHeaders(request.getAllResponseHeaders()) : null;
      var responseData = !config.responseType || config.responseType === 'text' ? request.responseText : request.response;
      var response = {
        data: responseData,
        status: request.status,
        statusText: request.statusText,
        headers: responseHeaders,
        config: config,
        request: request
      };

      settle(resolve, reject, response);

      // Clean up request
      request = null;
    };

    // Handle browser request cancellation (as opposed to a manual cancellation)
    request.onabort = function handleAbort() {
      if (!request) {
        return;
      }

      reject(createError('Request aborted', config, 'ECONNABORTED', request));

      // Clean up request
      request = null;
    };

    // Handle low level network errors
    request.onerror = function handleError() {
      // Real errors are hidden from us by the browser
      // onerror should only fire if it's a network error
      reject(createError('Network Error', config, null, request));

      // Clean up request
      request = null;
    };

    // Handle timeout
    request.ontimeout = function handleTimeout() {
      var timeoutErrorMessage = 'timeout of ' + config.timeout + 'ms exceeded';
      if (config.timeoutErrorMessage) {
        timeoutErrorMessage = config.timeoutErrorMessage;
      }
      reject(createError(timeoutErrorMessage, config, 'ECONNABORTED',
        request));

      // Clean up request
      request = null;
    };

    // Add xsrf header
    // This is only done if running in a standard browser environment.
    // Specifically not if we're in a web worker, or react-native.
    if (utils.isStandardBrowserEnv()) {
      // Add xsrf header
      var xsrfValue = (config.withCredentials || isURLSameOrigin(fullPath)) && config.xsrfCookieName ?
        cookies.read(config.xsrfCookieName) :
        undefined;

      if (xsrfValue) {
        requestHeaders[config.xsrfHeaderName] = xsrfValue;
      }
    }

    // Add headers to the request
    if ('setRequestHeader' in request) {
      utils.forEach(requestHeaders, function setRequestHeader(val, key) {
        if (typeof requestData === 'undefined' && key.toLowerCase() === 'content-type') {
          // Remove Content-Type if data is undefined
          delete requestHeaders[key];
        } else {
          // Otherwise add header to the request
          request.setRequestHeader(key, val);
        }
      });
    }

    // Add withCredentials to request if needed
    if (!utils.isUndefined(config.withCredentials)) {
      request.withCredentials = !!config.withCredentials;
    }

    // Add responseType to request if needed
    if (config.responseType) {
      try {
        request.responseType = config.responseType;
      } catch (e) {
        // Expected DOMException thrown by browsers not compatible XMLHttpRequest Level 2.
        // But, this can be suppressed for 'json' type as it can be parsed by default 'transformResponse' function.
        if (config.responseType !== 'json') {
          throw e;
        }
      }
    }

    // Handle progress if needed
    if (typeof config.onDownloadProgress === 'function') {
      request.addEventListener('progress', config.onDownloadProgress);
    }

    // Not all browsers support upload events
    if (typeof config.onUploadProgress === 'function' && request.upload) {
      request.upload.addEventListener('progress', config.onUploadProgress);
    }

    if (config.cancelToken) {
      // Handle cancellation
      config.cancelToken.promise.then(function onCanceled(cancel) {
        if (!request) {
          return;
        }

        request.abort();
        reject(cancel);
        // Clean up request
        request = null;
      });
    }

    if (!requestData) {
      requestData = null;
    }

    // Send the request
    request.send(requestData);
  });
};


/***/ }),

/***/ "./node_modules/axios/lib/axios.js":
/*!*****************************************!*\
  !*** ./node_modules/axios/lib/axios.js ***!
  \*****************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./utils */ "./node_modules/axios/lib/utils.js");
var bind = __webpack_require__(/*! ./helpers/bind */ "./node_modules/axios/lib/helpers/bind.js");
var Axios = __webpack_require__(/*! ./core/Axios */ "./node_modules/axios/lib/core/Axios.js");
var mergeConfig = __webpack_require__(/*! ./core/mergeConfig */ "./node_modules/axios/lib/core/mergeConfig.js");
var defaults = __webpack_require__(/*! ./defaults */ "./node_modules/axios/lib/defaults.js");

/**
 * Create an instance of Axios
 *
 * @param {Object} defaultConfig The default config for the instance
 * @return {Axios} A new instance of Axios
 */
function createInstance(defaultConfig) {
  var context = new Axios(defaultConfig);
  var instance = bind(Axios.prototype.request, context);

  // Copy axios.prototype to instance
  utils.extend(instance, Axios.prototype, context);

  // Copy context to instance
  utils.extend(instance, context);

  return instance;
}

// Create the default instance to be exported
var axios = createInstance(defaults);

// Expose Axios class to allow class inheritance
axios.Axios = Axios;

// Factory for creating new instances
axios.create = function create(instanceConfig) {
  return createInstance(mergeConfig(axios.defaults, instanceConfig));
};

// Expose Cancel & CancelToken
axios.Cancel = __webpack_require__(/*! ./cancel/Cancel */ "./node_modules/axios/lib/cancel/Cancel.js");
axios.CancelToken = __webpack_require__(/*! ./cancel/CancelToken */ "./node_modules/axios/lib/cancel/CancelToken.js");
axios.isCancel = __webpack_require__(/*! ./cancel/isCancel */ "./node_modules/axios/lib/cancel/isCancel.js");

// Expose all/spread
axios.all = function all(promises) {
  return Promise.all(promises);
};
axios.spread = __webpack_require__(/*! ./helpers/spread */ "./node_modules/axios/lib/helpers/spread.js");

// Expose isAxiosError
axios.isAxiosError = __webpack_require__(/*! ./helpers/isAxiosError */ "./node_modules/axios/lib/helpers/isAxiosError.js");

module.exports = axios;

// Allow use of default import syntax in TypeScript
module.exports.default = axios;


/***/ }),

/***/ "./node_modules/axios/lib/cancel/Cancel.js":
/*!*************************************************!*\
  !*** ./node_modules/axios/lib/cancel/Cancel.js ***!
  \*************************************************/
/***/ ((module) => {

"use strict";


/**
 * A `Cancel` is an object that is thrown when an operation is canceled.
 *
 * @class
 * @param {string=} message The message.
 */
function Cancel(message) {
  this.message = message;
}

Cancel.prototype.toString = function toString() {
  return 'Cancel' + (this.message ? ': ' + this.message : '');
};

Cancel.prototype.__CANCEL__ = true;

module.exports = Cancel;


/***/ }),

/***/ "./node_modules/axios/lib/cancel/CancelToken.js":
/*!******************************************************!*\
  !*** ./node_modules/axios/lib/cancel/CancelToken.js ***!
  \******************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var Cancel = __webpack_require__(/*! ./Cancel */ "./node_modules/axios/lib/cancel/Cancel.js");

/**
 * A `CancelToken` is an object that can be used to request cancellation of an operation.
 *
 * @class
 * @param {Function} executor The executor function.
 */
function CancelToken(executor) {
  if (typeof executor !== 'function') {
    throw new TypeError('executor must be a function.');
  }

  var resolvePromise;
  this.promise = new Promise(function promiseExecutor(resolve) {
    resolvePromise = resolve;
  });

  var token = this;
  executor(function cancel(message) {
    if (token.reason) {
      // Cancellation has already been requested
      return;
    }

    token.reason = new Cancel(message);
    resolvePromise(token.reason);
  });
}

/**
 * Throws a `Cancel` if cancellation has been requested.
 */
CancelToken.prototype.throwIfRequested = function throwIfRequested() {
  if (this.reason) {
    throw this.reason;
  }
};

/**
 * Returns an object that contains a new `CancelToken` and a function that, when called,
 * cancels the `CancelToken`.
 */
CancelToken.source = function source() {
  var cancel;
  var token = new CancelToken(function executor(c) {
    cancel = c;
  });
  return {
    token: token,
    cancel: cancel
  };
};

module.exports = CancelToken;


/***/ }),

/***/ "./node_modules/axios/lib/cancel/isCancel.js":
/*!***************************************************!*\
  !*** ./node_modules/axios/lib/cancel/isCancel.js ***!
  \***************************************************/
/***/ ((module) => {

"use strict";


module.exports = function isCancel(value) {
  return !!(value && value.__CANCEL__);
};


/***/ }),

/***/ "./node_modules/axios/lib/core/Axios.js":
/*!**********************************************!*\
  !*** ./node_modules/axios/lib/core/Axios.js ***!
  \**********************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./../utils */ "./node_modules/axios/lib/utils.js");
var buildURL = __webpack_require__(/*! ../helpers/buildURL */ "./node_modules/axios/lib/helpers/buildURL.js");
var InterceptorManager = __webpack_require__(/*! ./InterceptorManager */ "./node_modules/axios/lib/core/InterceptorManager.js");
var dispatchRequest = __webpack_require__(/*! ./dispatchRequest */ "./node_modules/axios/lib/core/dispatchRequest.js");
var mergeConfig = __webpack_require__(/*! ./mergeConfig */ "./node_modules/axios/lib/core/mergeConfig.js");

/**
 * Create a new instance of Axios
 *
 * @param {Object} instanceConfig The default config for the instance
 */
function Axios(instanceConfig) {
  this.defaults = instanceConfig;
  this.interceptors = {
    request: new InterceptorManager(),
    response: new InterceptorManager()
  };
}

/**
 * Dispatch a request
 *
 * @param {Object} config The config specific for this request (merged with this.defaults)
 */
Axios.prototype.request = function request(config) {
  /*eslint no-param-reassign:0*/
  // Allow for axios('example/url'[, config]) a la fetch API
  if (typeof config === 'string') {
    config = arguments[1] || {};
    config.url = arguments[0];
  } else {
    config = config || {};
  }

  config = mergeConfig(this.defaults, config);

  // Set config.method
  if (config.method) {
    config.method = config.method.toLowerCase();
  } else if (this.defaults.method) {
    config.method = this.defaults.method.toLowerCase();
  } else {
    config.method = 'get';
  }

  // Hook up interceptors middleware
  var chain = [dispatchRequest, undefined];
  var promise = Promise.resolve(config);

  this.interceptors.request.forEach(function unshiftRequestInterceptors(interceptor) {
    chain.unshift(interceptor.fulfilled, interceptor.rejected);
  });

  this.interceptors.response.forEach(function pushResponseInterceptors(interceptor) {
    chain.push(interceptor.fulfilled, interceptor.rejected);
  });

  while (chain.length) {
    promise = promise.then(chain.shift(), chain.shift());
  }

  return promise;
};

Axios.prototype.getUri = function getUri(config) {
  config = mergeConfig(this.defaults, config);
  return buildURL(config.url, config.params, config.paramsSerializer).replace(/^\?/, '');
};

// Provide aliases for supported request methods
utils.forEach(['delete', 'get', 'head', 'options'], function forEachMethodNoData(method) {
  /*eslint func-names:0*/
  Axios.prototype[method] = function(url, config) {
    return this.request(mergeConfig(config || {}, {
      method: method,
      url: url,
      data: (config || {}).data
    }));
  };
});

utils.forEach(['post', 'put', 'patch'], function forEachMethodWithData(method) {
  /*eslint func-names:0*/
  Axios.prototype[method] = function(url, data, config) {
    return this.request(mergeConfig(config || {}, {
      method: method,
      url: url,
      data: data
    }));
  };
});

module.exports = Axios;


/***/ }),

/***/ "./node_modules/axios/lib/core/InterceptorManager.js":
/*!***********************************************************!*\
  !*** ./node_modules/axios/lib/core/InterceptorManager.js ***!
  \***********************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./../utils */ "./node_modules/axios/lib/utils.js");

function InterceptorManager() {
  this.handlers = [];
}

/**
 * Add a new interceptor to the stack
 *
 * @param {Function} fulfilled The function to handle `then` for a `Promise`
 * @param {Function} rejected The function to handle `reject` for a `Promise`
 *
 * @return {Number} An ID used to remove interceptor later
 */
InterceptorManager.prototype.use = function use(fulfilled, rejected) {
  this.handlers.push({
    fulfilled: fulfilled,
    rejected: rejected
  });
  return this.handlers.length - 1;
};

/**
 * Remove an interceptor from the stack
 *
 * @param {Number} id The ID that was returned by `use`
 */
InterceptorManager.prototype.eject = function eject(id) {
  if (this.handlers[id]) {
    this.handlers[id] = null;
  }
};

/**
 * Iterate over all the registered interceptors
 *
 * This method is particularly useful for skipping over any
 * interceptors that may have become `null` calling `eject`.
 *
 * @param {Function} fn The function to call for each interceptor
 */
InterceptorManager.prototype.forEach = function forEach(fn) {
  utils.forEach(this.handlers, function forEachHandler(h) {
    if (h !== null) {
      fn(h);
    }
  });
};

module.exports = InterceptorManager;


/***/ }),

/***/ "./node_modules/axios/lib/core/buildFullPath.js":
/*!******************************************************!*\
  !*** ./node_modules/axios/lib/core/buildFullPath.js ***!
  \******************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var isAbsoluteURL = __webpack_require__(/*! ../helpers/isAbsoluteURL */ "./node_modules/axios/lib/helpers/isAbsoluteURL.js");
var combineURLs = __webpack_require__(/*! ../helpers/combineURLs */ "./node_modules/axios/lib/helpers/combineURLs.js");

/**
 * Creates a new URL by combining the baseURL with the requestedURL,
 * only when the requestedURL is not already an absolute URL.
 * If the requestURL is absolute, this function returns the requestedURL untouched.
 *
 * @param {string} baseURL The base URL
 * @param {string} requestedURL Absolute or relative URL to combine
 * @returns {string} The combined full path
 */
module.exports = function buildFullPath(baseURL, requestedURL) {
  if (baseURL && !isAbsoluteURL(requestedURL)) {
    return combineURLs(baseURL, requestedURL);
  }
  return requestedURL;
};


/***/ }),

/***/ "./node_modules/axios/lib/core/createError.js":
/*!****************************************************!*\
  !*** ./node_modules/axios/lib/core/createError.js ***!
  \****************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var enhanceError = __webpack_require__(/*! ./enhanceError */ "./node_modules/axios/lib/core/enhanceError.js");

/**
 * Create an Error with the specified message, config, error code, request and response.
 *
 * @param {string} message The error message.
 * @param {Object} config The config.
 * @param {string} [code] The error code (for example, 'ECONNABORTED').
 * @param {Object} [request] The request.
 * @param {Object} [response] The response.
 * @returns {Error} The created error.
 */
module.exports = function createError(message, config, code, request, response) {
  var error = new Error(message);
  return enhanceError(error, config, code, request, response);
};


/***/ }),

/***/ "./node_modules/axios/lib/core/dispatchRequest.js":
/*!********************************************************!*\
  !*** ./node_modules/axios/lib/core/dispatchRequest.js ***!
  \********************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./../utils */ "./node_modules/axios/lib/utils.js");
var transformData = __webpack_require__(/*! ./transformData */ "./node_modules/axios/lib/core/transformData.js");
var isCancel = __webpack_require__(/*! ../cancel/isCancel */ "./node_modules/axios/lib/cancel/isCancel.js");
var defaults = __webpack_require__(/*! ../defaults */ "./node_modules/axios/lib/defaults.js");

/**
 * Throws a `Cancel` if cancellation has been requested.
 */
function throwIfCancellationRequested(config) {
  if (config.cancelToken) {
    config.cancelToken.throwIfRequested();
  }
}

/**
 * Dispatch a request to the server using the configured adapter.
 *
 * @param {object} config The config that is to be used for the request
 * @returns {Promise} The Promise to be fulfilled
 */
module.exports = function dispatchRequest(config) {
  throwIfCancellationRequested(config);

  // Ensure headers exist
  config.headers = config.headers || {};

  // Transform request data
  config.data = transformData(
    config.data,
    config.headers,
    config.transformRequest
  );

  // Flatten headers
  config.headers = utils.merge(
    config.headers.common || {},
    config.headers[config.method] || {},
    config.headers
  );

  utils.forEach(
    ['delete', 'get', 'head', 'post', 'put', 'patch', 'common'],
    function cleanHeaderConfig(method) {
      delete config.headers[method];
    }
  );

  var adapter = config.adapter || defaults.adapter;

  return adapter(config).then(function onAdapterResolution(response) {
    throwIfCancellationRequested(config);

    // Transform response data
    response.data = transformData(
      response.data,
      response.headers,
      config.transformResponse
    );

    return response;
  }, function onAdapterRejection(reason) {
    if (!isCancel(reason)) {
      throwIfCancellationRequested(config);

      // Transform response data
      if (reason && reason.response) {
        reason.response.data = transformData(
          reason.response.data,
          reason.response.headers,
          config.transformResponse
        );
      }
    }

    return Promise.reject(reason);
  });
};


/***/ }),

/***/ "./node_modules/axios/lib/core/enhanceError.js":
/*!*****************************************************!*\
  !*** ./node_modules/axios/lib/core/enhanceError.js ***!
  \*****************************************************/
/***/ ((module) => {

"use strict";


/**
 * Update an Error with the specified config, error code, and response.
 *
 * @param {Error} error The error to update.
 * @param {Object} config The config.
 * @param {string} [code] The error code (for example, 'ECONNABORTED').
 * @param {Object} [request] The request.
 * @param {Object} [response] The response.
 * @returns {Error} The error.
 */
module.exports = function enhanceError(error, config, code, request, response) {
  error.config = config;
  if (code) {
    error.code = code;
  }

  error.request = request;
  error.response = response;
  error.isAxiosError = true;

  error.toJSON = function toJSON() {
    return {
      // Standard
      message: this.message,
      name: this.name,
      // Microsoft
      description: this.description,
      number: this.number,
      // Mozilla
      fileName: this.fileName,
      lineNumber: this.lineNumber,
      columnNumber: this.columnNumber,
      stack: this.stack,
      // Axios
      config: this.config,
      code: this.code
    };
  };
  return error;
};


/***/ }),

/***/ "./node_modules/axios/lib/core/mergeConfig.js":
/*!****************************************************!*\
  !*** ./node_modules/axios/lib/core/mergeConfig.js ***!
  \****************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ../utils */ "./node_modules/axios/lib/utils.js");

/**
 * Config-specific merge-function which creates a new config-object
 * by merging two configuration objects together.
 *
 * @param {Object} config1
 * @param {Object} config2
 * @returns {Object} New object resulting from merging config2 to config1
 */
module.exports = function mergeConfig(config1, config2) {
  // eslint-disable-next-line no-param-reassign
  config2 = config2 || {};
  var config = {};

  var valueFromConfig2Keys = ['url', 'method', 'data'];
  var mergeDeepPropertiesKeys = ['headers', 'auth', 'proxy', 'params'];
  var defaultToConfig2Keys = [
    'baseURL', 'transformRequest', 'transformResponse', 'paramsSerializer',
    'timeout', 'timeoutMessage', 'withCredentials', 'adapter', 'responseType', 'xsrfCookieName',
    'xsrfHeaderName', 'onUploadProgress', 'onDownloadProgress', 'decompress',
    'maxContentLength', 'maxBodyLength', 'maxRedirects', 'transport', 'httpAgent',
    'httpsAgent', 'cancelToken', 'socketPath', 'responseEncoding'
  ];
  var directMergeKeys = ['validateStatus'];

  function getMergedValue(target, source) {
    if (utils.isPlainObject(target) && utils.isPlainObject(source)) {
      return utils.merge(target, source);
    } else if (utils.isPlainObject(source)) {
      return utils.merge({}, source);
    } else if (utils.isArray(source)) {
      return source.slice();
    }
    return source;
  }

  function mergeDeepProperties(prop) {
    if (!utils.isUndefined(config2[prop])) {
      config[prop] = getMergedValue(config1[prop], config2[prop]);
    } else if (!utils.isUndefined(config1[prop])) {
      config[prop] = getMergedValue(undefined, config1[prop]);
    }
  }

  utils.forEach(valueFromConfig2Keys, function valueFromConfig2(prop) {
    if (!utils.isUndefined(config2[prop])) {
      config[prop] = getMergedValue(undefined, config2[prop]);
    }
  });

  utils.forEach(mergeDeepPropertiesKeys, mergeDeepProperties);

  utils.forEach(defaultToConfig2Keys, function defaultToConfig2(prop) {
    if (!utils.isUndefined(config2[prop])) {
      config[prop] = getMergedValue(undefined, config2[prop]);
    } else if (!utils.isUndefined(config1[prop])) {
      config[prop] = getMergedValue(undefined, config1[prop]);
    }
  });

  utils.forEach(directMergeKeys, function merge(prop) {
    if (prop in config2) {
      config[prop] = getMergedValue(config1[prop], config2[prop]);
    } else if (prop in config1) {
      config[prop] = getMergedValue(undefined, config1[prop]);
    }
  });

  var axiosKeys = valueFromConfig2Keys
    .concat(mergeDeepPropertiesKeys)
    .concat(defaultToConfig2Keys)
    .concat(directMergeKeys);

  var otherKeys = Object
    .keys(config1)
    .concat(Object.keys(config2))
    .filter(function filterAxiosKeys(key) {
      return axiosKeys.indexOf(key) === -1;
    });

  utils.forEach(otherKeys, mergeDeepProperties);

  return config;
};


/***/ }),

/***/ "./node_modules/axios/lib/core/settle.js":
/*!***********************************************!*\
  !*** ./node_modules/axios/lib/core/settle.js ***!
  \***********************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var createError = __webpack_require__(/*! ./createError */ "./node_modules/axios/lib/core/createError.js");

/**
 * Resolve or reject a Promise based on response status.
 *
 * @param {Function} resolve A function that resolves the promise.
 * @param {Function} reject A function that rejects the promise.
 * @param {object} response The response.
 */
module.exports = function settle(resolve, reject, response) {
  var validateStatus = response.config.validateStatus;
  if (!response.status || !validateStatus || validateStatus(response.status)) {
    resolve(response);
  } else {
    reject(createError(
      'Request failed with status code ' + response.status,
      response.config,
      null,
      response.request,
      response
    ));
  }
};


/***/ }),

/***/ "./node_modules/axios/lib/core/transformData.js":
/*!******************************************************!*\
  !*** ./node_modules/axios/lib/core/transformData.js ***!
  \******************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./../utils */ "./node_modules/axios/lib/utils.js");

/**
 * Transform the data for a request or a response
 *
 * @param {Object|String} data The data to be transformed
 * @param {Array} headers The headers for the request or response
 * @param {Array|Function} fns A single function or Array of functions
 * @returns {*} The resulting transformed data
 */
module.exports = function transformData(data, headers, fns) {
  /*eslint no-param-reassign:0*/
  utils.forEach(fns, function transform(fn) {
    data = fn(data, headers);
  });

  return data;
};


/***/ }),

/***/ "./node_modules/axios/lib/defaults.js":
/*!********************************************!*\
  !*** ./node_modules/axios/lib/defaults.js ***!
  \********************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";
/* provided dependency */ var process = __webpack_require__(/*! process/browser */ "./node_modules/process/browser.js");


var utils = __webpack_require__(/*! ./utils */ "./node_modules/axios/lib/utils.js");
var normalizeHeaderName = __webpack_require__(/*! ./helpers/normalizeHeaderName */ "./node_modules/axios/lib/helpers/normalizeHeaderName.js");

var DEFAULT_CONTENT_TYPE = {
  'Content-Type': 'application/x-www-form-urlencoded'
};

function setContentTypeIfUnset(headers, value) {
  if (!utils.isUndefined(headers) && utils.isUndefined(headers['Content-Type'])) {
    headers['Content-Type'] = value;
  }
}

function getDefaultAdapter() {
  var adapter;
  if (typeof XMLHttpRequest !== 'undefined') {
    // For browsers use XHR adapter
    adapter = __webpack_require__(/*! ./adapters/xhr */ "./node_modules/axios/lib/adapters/xhr.js");
  } else if (typeof process !== 'undefined' && Object.prototype.toString.call(process) === '[object process]') {
    // For node use HTTP adapter
    adapter = __webpack_require__(/*! ./adapters/http */ "./node_modules/axios/lib/adapters/xhr.js");
  }
  return adapter;
}

var defaults = {
  adapter: getDefaultAdapter(),

  transformRequest: [function transformRequest(data, headers) {
    normalizeHeaderName(headers, 'Accept');
    normalizeHeaderName(headers, 'Content-Type');
    if (utils.isFormData(data) ||
      utils.isArrayBuffer(data) ||
      utils.isBuffer(data) ||
      utils.isStream(data) ||
      utils.isFile(data) ||
      utils.isBlob(data)
    ) {
      return data;
    }
    if (utils.isArrayBufferView(data)) {
      return data.buffer;
    }
    if (utils.isURLSearchParams(data)) {
      setContentTypeIfUnset(headers, 'application/x-www-form-urlencoded;charset=utf-8');
      return data.toString();
    }
    if (utils.isObject(data)) {
      setContentTypeIfUnset(headers, 'application/json;charset=utf-8');
      return JSON.stringify(data);
    }
    return data;
  }],

  transformResponse: [function transformResponse(data) {
    /*eslint no-param-reassign:0*/
    if (typeof data === 'string') {
      try {
        data = JSON.parse(data);
      } catch (e) { /* Ignore */ }
    }
    return data;
  }],

  /**
   * A timeout in milliseconds to abort a request. If set to 0 (default) a
   * timeout is not created.
   */
  timeout: 0,

  xsrfCookieName: 'XSRF-TOKEN',
  xsrfHeaderName: 'X-XSRF-TOKEN',

  maxContentLength: -1,
  maxBodyLength: -1,

  validateStatus: function validateStatus(status) {
    return status >= 200 && status < 300;
  }
};

defaults.headers = {
  common: {
    'Accept': 'application/json, text/plain, */*'
  }
};

utils.forEach(['delete', 'get', 'head'], function forEachMethodNoData(method) {
  defaults.headers[method] = {};
});

utils.forEach(['post', 'put', 'patch'], function forEachMethodWithData(method) {
  defaults.headers[method] = utils.merge(DEFAULT_CONTENT_TYPE);
});

module.exports = defaults;


/***/ }),

/***/ "./node_modules/axios/lib/helpers/bind.js":
/*!************************************************!*\
  !*** ./node_modules/axios/lib/helpers/bind.js ***!
  \************************************************/
/***/ ((module) => {

"use strict";


module.exports = function bind(fn, thisArg) {
  return function wrap() {
    var args = new Array(arguments.length);
    for (var i = 0; i < args.length; i++) {
      args[i] = arguments[i];
    }
    return fn.apply(thisArg, args);
  };
};


/***/ }),

/***/ "./node_modules/axios/lib/helpers/buildURL.js":
/*!****************************************************!*\
  !*** ./node_modules/axios/lib/helpers/buildURL.js ***!
  \****************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./../utils */ "./node_modules/axios/lib/utils.js");

function encode(val) {
  return encodeURIComponent(val).
    replace(/%3A/gi, ':').
    replace(/%24/g, '$').
    replace(/%2C/gi, ',').
    replace(/%20/g, '+').
    replace(/%5B/gi, '[').
    replace(/%5D/gi, ']');
}

/**
 * Build a URL by appending params to the end
 *
 * @param {string} url The base of the url (e.g., http://www.google.com)
 * @param {object} [params] The params to be appended
 * @returns {string} The formatted url
 */
module.exports = function buildURL(url, params, paramsSerializer) {
  /*eslint no-param-reassign:0*/
  if (!params) {
    return url;
  }

  var serializedParams;
  if (paramsSerializer) {
    serializedParams = paramsSerializer(params);
  } else if (utils.isURLSearchParams(params)) {
    serializedParams = params.toString();
  } else {
    var parts = [];

    utils.forEach(params, function serialize(val, key) {
      if (val === null || typeof val === 'undefined') {
        return;
      }

      if (utils.isArray(val)) {
        key = key + '[]';
      } else {
        val = [val];
      }

      utils.forEach(val, function parseValue(v) {
        if (utils.isDate(v)) {
          v = v.toISOString();
        } else if (utils.isObject(v)) {
          v = JSON.stringify(v);
        }
        parts.push(encode(key) + '=' + encode(v));
      });
    });

    serializedParams = parts.join('&');
  }

  if (serializedParams) {
    var hashmarkIndex = url.indexOf('#');
    if (hashmarkIndex !== -1) {
      url = url.slice(0, hashmarkIndex);
    }

    url += (url.indexOf('?') === -1 ? '?' : '&') + serializedParams;
  }

  return url;
};


/***/ }),

/***/ "./node_modules/axios/lib/helpers/combineURLs.js":
/*!*******************************************************!*\
  !*** ./node_modules/axios/lib/helpers/combineURLs.js ***!
  \*******************************************************/
/***/ ((module) => {

"use strict";


/**
 * Creates a new URL by combining the specified URLs
 *
 * @param {string} baseURL The base URL
 * @param {string} relativeURL The relative URL
 * @returns {string} The combined URL
 */
module.exports = function combineURLs(baseURL, relativeURL) {
  return relativeURL
    ? baseURL.replace(/\/+$/, '') + '/' + relativeURL.replace(/^\/+/, '')
    : baseURL;
};


/***/ }),

/***/ "./node_modules/axios/lib/helpers/cookies.js":
/*!***************************************************!*\
  !*** ./node_modules/axios/lib/helpers/cookies.js ***!
  \***************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./../utils */ "./node_modules/axios/lib/utils.js");

module.exports = (
  utils.isStandardBrowserEnv() ?

  // Standard browser envs support document.cookie
    (function standardBrowserEnv() {
      return {
        write: function write(name, value, expires, path, domain, secure) {
          var cookie = [];
          cookie.push(name + '=' + encodeURIComponent(value));

          if (utils.isNumber(expires)) {
            cookie.push('expires=' + new Date(expires).toGMTString());
          }

          if (utils.isString(path)) {
            cookie.push('path=' + path);
          }

          if (utils.isString(domain)) {
            cookie.push('domain=' + domain);
          }

          if (secure === true) {
            cookie.push('secure');
          }

          document.cookie = cookie.join('; ');
        },

        read: function read(name) {
          var match = document.cookie.match(new RegExp('(^|;\\s*)(' + name + ')=([^;]*)'));
          return (match ? decodeURIComponent(match[3]) : null);
        },

        remove: function remove(name) {
          this.write(name, '', Date.now() - 86400000);
        }
      };
    })() :

  // Non standard browser env (web workers, react-native) lack needed support.
    (function nonStandardBrowserEnv() {
      return {
        write: function write() {},
        read: function read() { return null; },
        remove: function remove() {}
      };
    })()
);


/***/ }),

/***/ "./node_modules/axios/lib/helpers/isAbsoluteURL.js":
/*!*********************************************************!*\
  !*** ./node_modules/axios/lib/helpers/isAbsoluteURL.js ***!
  \*********************************************************/
/***/ ((module) => {

"use strict";


/**
 * Determines whether the specified URL is absolute
 *
 * @param {string} url The URL to test
 * @returns {boolean} True if the specified URL is absolute, otherwise false
 */
module.exports = function isAbsoluteURL(url) {
  // A URL is considered absolute if it begins with "<scheme>://" or "//" (protocol-relative URL).
  // RFC 3986 defines scheme name as a sequence of characters beginning with a letter and followed
  // by any combination of letters, digits, plus, period, or hyphen.
  return /^([a-z][a-z\d\+\-\.]*:)?\/\//i.test(url);
};


/***/ }),

/***/ "./node_modules/axios/lib/helpers/isAxiosError.js":
/*!********************************************************!*\
  !*** ./node_modules/axios/lib/helpers/isAxiosError.js ***!
  \********************************************************/
/***/ ((module) => {

"use strict";


/**
 * Determines whether the payload is an error thrown by Axios
 *
 * @param {*} payload The value to test
 * @returns {boolean} True if the payload is an error thrown by Axios, otherwise false
 */
module.exports = function isAxiosError(payload) {
  return (typeof payload === 'object') && (payload.isAxiosError === true);
};


/***/ }),

/***/ "./node_modules/axios/lib/helpers/isURLSameOrigin.js":
/*!***********************************************************!*\
  !*** ./node_modules/axios/lib/helpers/isURLSameOrigin.js ***!
  \***********************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./../utils */ "./node_modules/axios/lib/utils.js");

module.exports = (
  utils.isStandardBrowserEnv() ?

  // Standard browser envs have full support of the APIs needed to test
  // whether the request URL is of the same origin as current location.
    (function standardBrowserEnv() {
      var msie = /(msie|trident)/i.test(navigator.userAgent);
      var urlParsingNode = document.createElement('a');
      var originURL;

      /**
    * Parse a URL to discover it's components
    *
    * @param {String} url The URL to be parsed
    * @returns {Object}
    */
      function resolveURL(url) {
        var href = url;

        if (msie) {
        // IE needs attribute set twice to normalize properties
          urlParsingNode.setAttribute('href', href);
          href = urlParsingNode.href;
        }

        urlParsingNode.setAttribute('href', href);

        // urlParsingNode provides the UrlUtils interface - http://url.spec.whatwg.org/#urlutils
        return {
          href: urlParsingNode.href,
          protocol: urlParsingNode.protocol ? urlParsingNode.protocol.replace(/:$/, '') : '',
          host: urlParsingNode.host,
          search: urlParsingNode.search ? urlParsingNode.search.replace(/^\?/, '') : '',
          hash: urlParsingNode.hash ? urlParsingNode.hash.replace(/^#/, '') : '',
          hostname: urlParsingNode.hostname,
          port: urlParsingNode.port,
          pathname: (urlParsingNode.pathname.charAt(0) === '/') ?
            urlParsingNode.pathname :
            '/' + urlParsingNode.pathname
        };
      }

      originURL = resolveURL(window.location.href);

      /**
    * Determine if a URL shares the same origin as the current location
    *
    * @param {String} requestURL The URL to test
    * @returns {boolean} True if URL shares the same origin, otherwise false
    */
      return function isURLSameOrigin(requestURL) {
        var parsed = (utils.isString(requestURL)) ? resolveURL(requestURL) : requestURL;
        return (parsed.protocol === originURL.protocol &&
            parsed.host === originURL.host);
      };
    })() :

  // Non standard browser envs (web workers, react-native) lack needed support.
    (function nonStandardBrowserEnv() {
      return function isURLSameOrigin() {
        return true;
      };
    })()
);


/***/ }),

/***/ "./node_modules/axios/lib/helpers/normalizeHeaderName.js":
/*!***************************************************************!*\
  !*** ./node_modules/axios/lib/helpers/normalizeHeaderName.js ***!
  \***************************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ../utils */ "./node_modules/axios/lib/utils.js");

module.exports = function normalizeHeaderName(headers, normalizedName) {
  utils.forEach(headers, function processHeader(value, name) {
    if (name !== normalizedName && name.toUpperCase() === normalizedName.toUpperCase()) {
      headers[normalizedName] = value;
      delete headers[name];
    }
  });
};


/***/ }),

/***/ "./node_modules/axios/lib/helpers/parseHeaders.js":
/*!********************************************************!*\
  !*** ./node_modules/axios/lib/helpers/parseHeaders.js ***!
  \********************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var utils = __webpack_require__(/*! ./../utils */ "./node_modules/axios/lib/utils.js");

// Headers whose duplicates are ignored by node
// c.f. https://nodejs.org/api/http.html#http_message_headers
var ignoreDuplicateOf = [
  'age', 'authorization', 'content-length', 'content-type', 'etag',
  'expires', 'from', 'host', 'if-modified-since', 'if-unmodified-since',
  'last-modified', 'location', 'max-forwards', 'proxy-authorization',
  'referer', 'retry-after', 'user-agent'
];

/**
 * Parse headers into an object
 *
 * ```
 * Date: Wed, 27 Aug 2014 08:58:49 GMT
 * Content-Type: application/json
 * Connection: keep-alive
 * Transfer-Encoding: chunked
 * ```
 *
 * @param {String} headers Headers needing to be parsed
 * @returns {Object} Headers parsed into an object
 */
module.exports = function parseHeaders(headers) {
  var parsed = {};
  var key;
  var val;
  var i;

  if (!headers) { return parsed; }

  utils.forEach(headers.split('\n'), function parser(line) {
    i = line.indexOf(':');
    key = utils.trim(line.substr(0, i)).toLowerCase();
    val = utils.trim(line.substr(i + 1));

    if (key) {
      if (parsed[key] && ignoreDuplicateOf.indexOf(key) >= 0) {
        return;
      }
      if (key === 'set-cookie') {
        parsed[key] = (parsed[key] ? parsed[key] : []).concat([val]);
      } else {
        parsed[key] = parsed[key] ? parsed[key] + ', ' + val : val;
      }
    }
  });

  return parsed;
};


/***/ }),

/***/ "./node_modules/axios/lib/helpers/spread.js":
/*!**************************************************!*\
  !*** ./node_modules/axios/lib/helpers/spread.js ***!
  \**************************************************/
/***/ ((module) => {

"use strict";


/**
 * Syntactic sugar for invoking a function and expanding an array for arguments.
 *
 * Common use case would be to use `Function.prototype.apply`.
 *
 *  ```js
 *  function f(x, y, z) {}
 *  var args = [1, 2, 3];
 *  f.apply(null, args);
 *  ```
 *
 * With `spread` this example can be re-written.
 *
 *  ```js
 *  spread(function(x, y, z) {})([1, 2, 3]);
 *  ```
 *
 * @param {Function} callback
 * @returns {Function}
 */
module.exports = function spread(callback) {
  return function wrap(arr) {
    return callback.apply(null, arr);
  };
};


/***/ }),

/***/ "./node_modules/axios/lib/utils.js":
/*!*****************************************!*\
  !*** ./node_modules/axios/lib/utils.js ***!
  \*****************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var bind = __webpack_require__(/*! ./helpers/bind */ "./node_modules/axios/lib/helpers/bind.js");

/*global toString:true*/

// utils is a library of generic helper functions non-specific to axios

var toString = Object.prototype.toString;

/**
 * Determine if a value is an Array
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is an Array, otherwise false
 */
function isArray(val) {
  return toString.call(val) === '[object Array]';
}

/**
 * Determine if a value is undefined
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if the value is undefined, otherwise false
 */
function isUndefined(val) {
  return typeof val === 'undefined';
}

/**
 * Determine if a value is a Buffer
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a Buffer, otherwise false
 */
function isBuffer(val) {
  return val !== null && !isUndefined(val) && val.constructor !== null && !isUndefined(val.constructor)
    && typeof val.constructor.isBuffer === 'function' && val.constructor.isBuffer(val);
}

/**
 * Determine if a value is an ArrayBuffer
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is an ArrayBuffer, otherwise false
 */
function isArrayBuffer(val) {
  return toString.call(val) === '[object ArrayBuffer]';
}

/**
 * Determine if a value is a FormData
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is an FormData, otherwise false
 */
function isFormData(val) {
  return (typeof FormData !== 'undefined') && (val instanceof FormData);
}

/**
 * Determine if a value is a view on an ArrayBuffer
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a view on an ArrayBuffer, otherwise false
 */
function isArrayBufferView(val) {
  var result;
  if ((typeof ArrayBuffer !== 'undefined') && (ArrayBuffer.isView)) {
    result = ArrayBuffer.isView(val);
  } else {
    result = (val) && (val.buffer) && (val.buffer instanceof ArrayBuffer);
  }
  return result;
}

/**
 * Determine if a value is a String
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a String, otherwise false
 */
function isString(val) {
  return typeof val === 'string';
}

/**
 * Determine if a value is a Number
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a Number, otherwise false
 */
function isNumber(val) {
  return typeof val === 'number';
}

/**
 * Determine if a value is an Object
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is an Object, otherwise false
 */
function isObject(val) {
  return val !== null && typeof val === 'object';
}

/**
 * Determine if a value is a plain Object
 *
 * @param {Object} val The value to test
 * @return {boolean} True if value is a plain Object, otherwise false
 */
function isPlainObject(val) {
  if (toString.call(val) !== '[object Object]') {
    return false;
  }

  var prototype = Object.getPrototypeOf(val);
  return prototype === null || prototype === Object.prototype;
}

/**
 * Determine if a value is a Date
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a Date, otherwise false
 */
function isDate(val) {
  return toString.call(val) === '[object Date]';
}

/**
 * Determine if a value is a File
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a File, otherwise false
 */
function isFile(val) {
  return toString.call(val) === '[object File]';
}

/**
 * Determine if a value is a Blob
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a Blob, otherwise false
 */
function isBlob(val) {
  return toString.call(val) === '[object Blob]';
}

/**
 * Determine if a value is a Function
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a Function, otherwise false
 */
function isFunction(val) {
  return toString.call(val) === '[object Function]';
}

/**
 * Determine if a value is a Stream
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a Stream, otherwise false
 */
function isStream(val) {
  return isObject(val) && isFunction(val.pipe);
}

/**
 * Determine if a value is a URLSearchParams object
 *
 * @param {Object} val The value to test
 * @returns {boolean} True if value is a URLSearchParams object, otherwise false
 */
function isURLSearchParams(val) {
  return typeof URLSearchParams !== 'undefined' && val instanceof URLSearchParams;
}

/**
 * Trim excess whitespace off the beginning and end of a string
 *
 * @param {String} str The String to trim
 * @returns {String} The String freed of excess whitespace
 */
function trim(str) {
  return str.replace(/^\s*/, '').replace(/\s*$/, '');
}

/**
 * Determine if we're running in a standard browser environment
 *
 * This allows axios to run in a web worker, and react-native.
 * Both environments support XMLHttpRequest, but not fully standard globals.
 *
 * web workers:
 *  typeof window -> undefined
 *  typeof document -> undefined
 *
 * react-native:
 *  navigator.product -> 'ReactNative'
 * nativescript
 *  navigator.product -> 'NativeScript' or 'NS'
 */
function isStandardBrowserEnv() {
  if (typeof navigator !== 'undefined' && (navigator.product === 'ReactNative' ||
                                           navigator.product === 'NativeScript' ||
                                           navigator.product === 'NS')) {
    return false;
  }
  return (
    typeof window !== 'undefined' &&
    typeof document !== 'undefined'
  );
}

/**
 * Iterate over an Array or an Object invoking a function for each item.
 *
 * If `obj` is an Array callback will be called passing
 * the value, index, and complete array for each item.
 *
 * If 'obj' is an Object callback will be called passing
 * the value, key, and complete object for each property.
 *
 * @param {Object|Array} obj The object to iterate
 * @param {Function} fn The callback to invoke for each item
 */
function forEach(obj, fn) {
  // Don't bother if no value provided
  if (obj === null || typeof obj === 'undefined') {
    return;
  }

  // Force an array if not already something iterable
  if (typeof obj !== 'object') {
    /*eslint no-param-reassign:0*/
    obj = [obj];
  }

  if (isArray(obj)) {
    // Iterate over array values
    for (var i = 0, l = obj.length; i < l; i++) {
      fn.call(null, obj[i], i, obj);
    }
  } else {
    // Iterate over object keys
    for (var key in obj) {
      if (Object.prototype.hasOwnProperty.call(obj, key)) {
        fn.call(null, obj[key], key, obj);
      }
    }
  }
}

/**
 * Accepts varargs expecting each argument to be an object, then
 * immutably merges the properties of each object and returns result.
 *
 * When multiple objects contain the same key the later object in
 * the arguments list will take precedence.
 *
 * Example:
 *
 * ```js
 * var result = merge({foo: 123}, {foo: 456});
 * console.log(result.foo); // outputs 456
 * ```
 *
 * @param {Object} obj1 Object to merge
 * @returns {Object} Result of all merge properties
 */
function merge(/* obj1, obj2, obj3, ... */) {
  var result = {};
  function assignValue(val, key) {
    if (isPlainObject(result[key]) && isPlainObject(val)) {
      result[key] = merge(result[key], val);
    } else if (isPlainObject(val)) {
      result[key] = merge({}, val);
    } else if (isArray(val)) {
      result[key] = val.slice();
    } else {
      result[key] = val;
    }
  }

  for (var i = 0, l = arguments.length; i < l; i++) {
    forEach(arguments[i], assignValue);
  }
  return result;
}

/**
 * Extends object a by mutably adding to it the properties of object b.
 *
 * @param {Object} a The object to be extended
 * @param {Object} b The object to copy properties from
 * @param {Object} thisArg The object to bind function to
 * @return {Object} The resulting value of object a
 */
function extend(a, b, thisArg) {
  forEach(b, function assignValue(val, key) {
    if (thisArg && typeof val === 'function') {
      a[key] = bind(val, thisArg);
    } else {
      a[key] = val;
    }
  });
  return a;
}

/**
 * Remove byte order marker. This catches EF BB BF (the UTF-8 BOM)
 *
 * @param {string} content with BOM
 * @return {string} content value without BOM
 */
function stripBOM(content) {
  if (content.charCodeAt(0) === 0xFEFF) {
    content = content.slice(1);
  }
  return content;
}

module.exports = {
  isArray: isArray,
  isArrayBuffer: isArrayBuffer,
  isBuffer: isBuffer,
  isFormData: isFormData,
  isArrayBufferView: isArrayBufferView,
  isString: isString,
  isNumber: isNumber,
  isObject: isObject,
  isPlainObject: isPlainObject,
  isUndefined: isUndefined,
  isDate: isDate,
  isFile: isFile,
  isBlob: isBlob,
  isFunction: isFunction,
  isStream: isStream,
  isURLSearchParams: isURLSearchParams,
  isStandardBrowserEnv: isStandardBrowserEnv,
  forEach: forEach,
  merge: merge,
  extend: extend,
  trim: trim,
  stripBOM: stripBOM
};


/***/ }),

/***/ "./node_modules/codemirror/src/display/focus.js":
/*!******************************************************!*\
  !*** ./node_modules/codemirror/src/display/focus.js ***!
  \******************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "ensureFocus": () => (/* binding */ ensureFocus),
/* harmony export */   "delayBlurEvent": () => (/* binding */ delayBlurEvent),
/* harmony export */   "onFocus": () => (/* binding */ onFocus),
/* harmony export */   "onBlur": () => (/* binding */ onBlur)
/* harmony export */ });
/* harmony import */ var _selection_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./selection.js */ "./node_modules/codemirror/src/display/selection.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");





function ensureFocus(cm) {
  if (!cm.hasFocus()) {
    cm.display.input.focus()
    if (!cm.state.focused) onFocus(cm)
  }
}

function delayBlurEvent(cm) {
  cm.state.delayingBlurEvent = true
  setTimeout(() => { if (cm.state.delayingBlurEvent) {
    cm.state.delayingBlurEvent = false
    if (cm.state.focused) onBlur(cm)
  } }, 100)
}

function onFocus(cm, e) {
  if (cm.state.delayingBlurEvent && !cm.state.draggingText) cm.state.delayingBlurEvent = false

  if (cm.options.readOnly == "nocursor") return
  if (!cm.state.focused) {
    (0,_util_event_js__WEBPACK_IMPORTED_MODULE_3__.signal)(cm, "focus", cm, e)
    cm.state.focused = true
    ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.addClass)(cm.display.wrapper, "CodeMirror-focused")
    // This test prevents this from firing when a context
    // menu is closed (since the input reset would kill the
    // select-all detection hack)
    if (!cm.curOp && cm.display.selForContextMenu != cm.doc.sel) {
      cm.display.input.reset()
      if (_util_browser_js__WEBPACK_IMPORTED_MODULE_1__.webkit) setTimeout(() => cm.display.input.reset(true), 20) // Issue #1730
    }
    cm.display.input.receivedFocus()
  }
  (0,_selection_js__WEBPACK_IMPORTED_MODULE_0__.restartBlink)(cm)
}
function onBlur(cm, e) {
  if (cm.state.delayingBlurEvent) return

  if (cm.state.focused) {
    (0,_util_event_js__WEBPACK_IMPORTED_MODULE_3__.signal)(cm, "blur", cm, e)
    cm.state.focused = false
    ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.rmClass)(cm.display.wrapper, "CodeMirror-focused")
  }
  clearInterval(cm.display.blinker)
  setTimeout(() => { if (!cm.state.focused) cm.display.shift = false }, 150)
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/highlight_worker.js":
/*!*****************************************************************!*\
  !*** ./node_modules/codemirror/src/display/highlight_worker.js ***!
  \*****************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "startWorker": () => (/* binding */ startWorker)
/* harmony export */ });
/* harmony import */ var _line_highlight_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/highlight.js */ "./node_modules/codemirror/src/line/highlight.js");
/* harmony import */ var _modes_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../modes.js */ "./node_modules/codemirror/src/modes.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _operations_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ./operations.js */ "./node_modules/codemirror/src/display/operations.js");
/* harmony import */ var _view_tracking_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ./view_tracking.js */ "./node_modules/codemirror/src/display/view_tracking.js");







// HIGHLIGHT WORKER

function startWorker(cm, time) {
  if (cm.doc.highlightFrontier < cm.display.viewTo)
    cm.state.highlight.set(time, (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_2__.bind)(highlightWorker, cm))
}

function highlightWorker(cm) {
  let doc = cm.doc
  if (doc.highlightFrontier >= cm.display.viewTo) return
  let end = +new Date + cm.options.workTime
  let context = (0,_line_highlight_js__WEBPACK_IMPORTED_MODULE_0__.getContextBefore)(cm, doc.highlightFrontier)
  let changedLines = []

  doc.iter(context.line, Math.min(doc.first + doc.size, cm.display.viewTo + 500), line => {
    if (context.line >= cm.display.viewFrom) { // Visible
      let oldStyles = line.styles
      let resetState = line.text.length > cm.options.maxHighlightLength ? (0,_modes_js__WEBPACK_IMPORTED_MODULE_1__.copyState)(doc.mode, context.state) : null
      let highlighted = (0,_line_highlight_js__WEBPACK_IMPORTED_MODULE_0__.highlightLine)(cm, line, context, true)
      if (resetState) context.state = resetState
      line.styles = highlighted.styles
      let oldCls = line.styleClasses, newCls = highlighted.classes
      if (newCls) line.styleClasses = newCls
      else if (oldCls) line.styleClasses = null
      let ischange = !oldStyles || oldStyles.length != line.styles.length ||
        oldCls != newCls && (!oldCls || !newCls || oldCls.bgClass != newCls.bgClass || oldCls.textClass != newCls.textClass)
      for (let i = 0; !ischange && i < oldStyles.length; ++i) ischange = oldStyles[i] != line.styles[i]
      if (ischange) changedLines.push(context.line)
      line.stateAfter = context.save()
      context.nextLine()
    } else {
      if (line.text.length <= cm.options.maxHighlightLength)
        (0,_line_highlight_js__WEBPACK_IMPORTED_MODULE_0__.processLine)(cm, line.text, context)
      line.stateAfter = context.line % 5 == 0 ? context.save() : null
      context.nextLine()
    }
    if (+new Date > end) {
      startWorker(cm, cm.options.workDelay)
      return true
    }
  })
  doc.highlightFrontier = context.line
  doc.modeFrontier = Math.max(doc.modeFrontier, context.line)
  if (changedLines.length) (0,_operations_js__WEBPACK_IMPORTED_MODULE_3__.runInOp)(cm, () => {
    for (let i = 0; i < changedLines.length; i++)
      (0,_view_tracking_js__WEBPACK_IMPORTED_MODULE_4__.regLineChange)(cm, changedLines[i], "text")
  })
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/line_numbers.js":
/*!*************************************************************!*\
  !*** ./node_modules/codemirror/src/display/line_numbers.js ***!
  \*************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "alignHorizontally": () => (/* binding */ alignHorizontally),
/* harmony export */   "maybeUpdateLineNumberWidth": () => (/* binding */ maybeUpdateLineNumberWidth)
/* harmony export */ });
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _update_display_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ./update_display.js */ "./node_modules/codemirror/src/display/update_display.js");






// Re-align line numbers and gutter marks to compensate for
// horizontal scrolling.
function alignHorizontally(cm) {
  let display = cm.display, view = display.view
  if (!display.alignWidgets && (!display.gutters.firstChild || !cm.options.fixedGutter)) return
  let comp = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.compensateForHScroll)(display) - display.scroller.scrollLeft + cm.doc.scrollLeft
  let gutterW = display.gutters.offsetWidth, left = comp + "px"
  for (let i = 0; i < view.length; i++) if (!view[i].hidden) {
    if (cm.options.fixedGutter) {
      if (view[i].gutter)
        view[i].gutter.style.left = left
      if (view[i].gutterBackground)
        view[i].gutterBackground.style.left = left
    }
    let align = view[i].alignable
    if (align) for (let j = 0; j < align.length; j++)
      align[j].style.left = left
  }
  if (cm.options.fixedGutter)
    display.gutters.style.left = (comp + gutterW) + "px"
}

// Used to ensure that the line number gutter is still the right
// size for the current document size. Returns true when an update
// is needed.
function maybeUpdateLineNumberWidth(cm) {
  if (!cm.options.lineNumbers) return false
  let doc = cm.doc, last = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_0__.lineNumberFor)(cm.options, doc.first + doc.size - 1), display = cm.display
  if (last.length != display.lineNumChars) {
    let test = display.measure.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.elt)("div", [(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.elt)("div", last)],
                                               "CodeMirror-linenumber CodeMirror-gutter-elt"))
    let innerW = test.firstChild.offsetWidth, padding = test.offsetWidth - innerW
    display.lineGutter.style.width = ""
    display.lineNumInnerWidth = Math.max(innerW, display.lineGutter.offsetWidth - padding) + 1
    display.lineNumWidth = display.lineNumInnerWidth + padding
    display.lineNumChars = display.lineNumInnerWidth ? last.length : -1
    display.lineGutter.style.width = display.lineNumWidth + "px"
    ;(0,_update_display_js__WEBPACK_IMPORTED_MODULE_3__.updateGutterSpace)(cm.display)
    return true
  }
  return false
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/mode_state.js":
/*!***********************************************************!*\
  !*** ./node_modules/codemirror/src/display/mode_state.js ***!
  \***********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "loadMode": () => (/* binding */ loadMode),
/* harmony export */   "resetModeState": () => (/* binding */ resetModeState)
/* harmony export */ });
/* harmony import */ var _modes_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../modes.js */ "./node_modules/codemirror/src/modes.js");
/* harmony import */ var _highlight_worker_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ./highlight_worker.js */ "./node_modules/codemirror/src/display/highlight_worker.js");
/* harmony import */ var _view_tracking_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ./view_tracking.js */ "./node_modules/codemirror/src/display/view_tracking.js");





// Used to get the editor into a consistent state again when options change.

function loadMode(cm) {
  cm.doc.mode = (0,_modes_js__WEBPACK_IMPORTED_MODULE_0__.getMode)(cm.options, cm.doc.modeOption)
  resetModeState(cm)
}

function resetModeState(cm) {
  cm.doc.iter(line => {
    if (line.stateAfter) line.stateAfter = null
    if (line.styles) line.styles = null
  })
  cm.doc.modeFrontier = cm.doc.highlightFrontier = cm.doc.first
  ;(0,_highlight_worker_js__WEBPACK_IMPORTED_MODULE_1__.startWorker)(cm, 100)
  cm.state.modeGen++
  if (cm.curOp) (0,_view_tracking_js__WEBPACK_IMPORTED_MODULE_2__.regChange)(cm)
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/operations.js":
/*!***********************************************************!*\
  !*** ./node_modules/codemirror/src/display/operations.js ***!
  \***********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "startOperation": () => (/* binding */ startOperation),
/* harmony export */   "endOperation": () => (/* binding */ endOperation),
/* harmony export */   "runInOp": () => (/* binding */ runInOp),
/* harmony export */   "operation": () => (/* binding */ operation),
/* harmony export */   "methodOp": () => (/* binding */ methodOp),
/* harmony export */   "docMethodOp": () => (/* binding */ docMethodOp)
/* harmony export */ });
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_operation_group_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../util/operation_group.js */ "./node_modules/codemirror/src/util/operation_group.js");
/* harmony import */ var _focus_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ./focus.js */ "./node_modules/codemirror/src/display/focus.js");
/* harmony import */ var _scrollbars_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ./scrollbars.js */ "./node_modules/codemirror/src/display/scrollbars.js");
/* harmony import */ var _selection_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ./selection.js */ "./node_modules/codemirror/src/display/selection.js");
/* harmony import */ var _scrolling_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ./scrolling.js */ "./node_modules/codemirror/src/display/scrolling.js");
/* harmony import */ var _update_display_js__WEBPACK_IMPORTED_MODULE_10__ = __webpack_require__(/*! ./update_display.js */ "./node_modules/codemirror/src/display/update_display.js");
/* harmony import */ var _update_lines_js__WEBPACK_IMPORTED_MODULE_11__ = __webpack_require__(/*! ./update_lines.js */ "./node_modules/codemirror/src/display/update_lines.js");














// Operations are used to wrap a series of changes to the editor
// state in such a way that each change won't have to update the
// cursor and display (which would be awkward, slow, and
// error-prone). Instead, display updates are batched and then all
// combined and executed at once.

let nextOpId = 0
// Start a new operation.
function startOperation(cm) {
  cm.curOp = {
    cm: cm,
    viewChanged: false,      // Flag that indicates that lines might need to be redrawn
    startHeight: cm.doc.height, // Used to detect need to update scrollbar
    forceUpdate: false,      // Used to force a redraw
    updateInput: 0,       // Whether to reset the input textarea
    typing: false,           // Whether this reset should be careful to leave existing text (for compositing)
    changeObjs: null,        // Accumulated changes, for firing change events
    cursorActivityHandlers: null, // Set of handlers to fire cursorActivity on
    cursorActivityCalled: 0, // Tracks which cursorActivity handlers have been called already
    selectionChanged: false, // Whether the selection needs to be redrawn
    updateMaxLine: false,    // Set when the widest line needs to be determined anew
    scrollLeft: null, scrollTop: null, // Intermediate scroll position, not pushed to DOM yet
    scrollToPos: null,       // Used to scroll to a specific position
    focus: false,
    id: ++nextOpId,          // Unique ID
    markArrays: null         // Used by addMarkedSpan
  }
  ;(0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_5__.pushOperation)(cm.curOp)
}

// Finish an operation, updating the display and signalling delayed events
function endOperation(cm) {
  let op = cm.curOp
  if (op) (0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_5__.finishOperation)(op, group => {
    for (let i = 0; i < group.ops.length; i++)
      group.ops[i].cm.curOp = null
    endOperations(group)
  })
}

// The DOM updates done when an operation finishes are batched so
// that the minimum number of relayouts are required.
function endOperations(group) {
  let ops = group.ops
  for (let i = 0; i < ops.length; i++) // Read DOM
    endOperation_R1(ops[i])
  for (let i = 0; i < ops.length; i++) // Write DOM (maybe)
    endOperation_W1(ops[i])
  for (let i = 0; i < ops.length; i++) // Read DOM
    endOperation_R2(ops[i])
  for (let i = 0; i < ops.length; i++) // Write DOM (maybe)
    endOperation_W2(ops[i])
  for (let i = 0; i < ops.length; i++) // Read DOM
    endOperation_finish(ops[i])
}

function endOperation_R1(op) {
  let cm = op.cm, display = cm.display
  ;(0,_update_display_js__WEBPACK_IMPORTED_MODULE_10__.maybeClipScrollbars)(cm)
  if (op.updateMaxLine) (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_1__.findMaxLine)(cm)

  op.mustUpdate = op.viewChanged || op.forceUpdate || op.scrollTop != null ||
    op.scrollToPos && (op.scrollToPos.from.line < display.viewFrom ||
                       op.scrollToPos.to.line >= display.viewTo) ||
    display.maxLineChanged && cm.options.lineWrapping
  op.update = op.mustUpdate &&
    new _update_display_js__WEBPACK_IMPORTED_MODULE_10__.DisplayUpdate(cm, op.mustUpdate && {top: op.scrollTop, ensure: op.scrollToPos}, op.forceUpdate)
}

function endOperation_W1(op) {
  op.updatedDisplay = op.mustUpdate && (0,_update_display_js__WEBPACK_IMPORTED_MODULE_10__.updateDisplayIfNeeded)(op.cm, op.update)
}

function endOperation_R2(op) {
  let cm = op.cm, display = cm.display
  if (op.updatedDisplay) (0,_update_lines_js__WEBPACK_IMPORTED_MODULE_11__.updateHeightsInViewport)(cm)

  op.barMeasure = (0,_scrollbars_js__WEBPACK_IMPORTED_MODULE_7__.measureForScrollbars)(cm)

  // If the max line changed since it was last measured, measure it,
  // and ensure the document's width matches it.
  // updateDisplay_W2 will use these properties to do the actual resizing
  if (display.maxLineChanged && !cm.options.lineWrapping) {
    op.adjustWidthTo = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__.measureChar)(cm, display.maxLine, display.maxLine.text.length).left + 3
    cm.display.sizerWidth = op.adjustWidthTo
    op.barMeasure.scrollWidth =
      Math.max(display.scroller.clientWidth, display.sizer.offsetLeft + op.adjustWidthTo + (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__.scrollGap)(cm) + cm.display.barWidth)
    op.maxScrollLeft = Math.max(0, display.sizer.offsetLeft + op.adjustWidthTo - (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__.displayWidth)(cm))
  }

  if (op.updatedDisplay || op.selectionChanged)
    op.preparedSelection = display.input.prepareSelection()
}

function endOperation_W2(op) {
  let cm = op.cm

  if (op.adjustWidthTo != null) {
    cm.display.sizer.style.minWidth = op.adjustWidthTo + "px"
    if (op.maxScrollLeft < cm.doc.scrollLeft)
      (0,_scrolling_js__WEBPACK_IMPORTED_MODULE_9__.setScrollLeft)(cm, Math.min(cm.display.scroller.scrollLeft, op.maxScrollLeft), true)
    cm.display.maxLineChanged = false
  }

  let takeFocus = op.focus && op.focus == (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_4__.activeElt)()
  if (op.preparedSelection)
    cm.display.input.showSelection(op.preparedSelection, takeFocus)
  if (op.updatedDisplay || op.startHeight != cm.doc.height)
    (0,_scrollbars_js__WEBPACK_IMPORTED_MODULE_7__.updateScrollbars)(cm, op.barMeasure)
  if (op.updatedDisplay)
    (0,_update_display_js__WEBPACK_IMPORTED_MODULE_10__.setDocumentHeight)(cm, op.barMeasure)

  if (op.selectionChanged) (0,_selection_js__WEBPACK_IMPORTED_MODULE_8__.restartBlink)(cm)

  if (cm.state.focused && op.updateInput)
    cm.display.input.reset(op.typing)
  if (takeFocus) (0,_focus_js__WEBPACK_IMPORTED_MODULE_6__.ensureFocus)(op.cm)
}

function endOperation_finish(op) {
  let cm = op.cm, display = cm.display, doc = cm.doc

  if (op.updatedDisplay) (0,_update_display_js__WEBPACK_IMPORTED_MODULE_10__.postUpdateDisplay)(cm, op.update)

  // Abort mouse wheel delta measurement, when scrolling explicitly
  if (display.wheelStartX != null && (op.scrollTop != null || op.scrollLeft != null || op.scrollToPos))
    display.wheelStartX = display.wheelStartY = null

  // Propagate the scroll position to the actual DOM scroller
  if (op.scrollTop != null) (0,_scrolling_js__WEBPACK_IMPORTED_MODULE_9__.setScrollTop)(cm, op.scrollTop, op.forceScroll)

  if (op.scrollLeft != null) (0,_scrolling_js__WEBPACK_IMPORTED_MODULE_9__.setScrollLeft)(cm, op.scrollLeft, true, true)
  // If we need to scroll a specific position into view, do so.
  if (op.scrollToPos) {
    let rect = (0,_scrolling_js__WEBPACK_IMPORTED_MODULE_9__.scrollPosIntoView)(cm, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.clipPos)(doc, op.scrollToPos.from),
                                 (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.clipPos)(doc, op.scrollToPos.to), op.scrollToPos.margin)
    ;(0,_scrolling_js__WEBPACK_IMPORTED_MODULE_9__.maybeScrollWindow)(cm, rect)
  }

  // Fire events for markers that are hidden/unidden by editing or
  // undoing
  let hidden = op.maybeHiddenMarkers, unhidden = op.maybeUnhiddenMarkers
  if (hidden) for (let i = 0; i < hidden.length; ++i)
    if (!hidden[i].lines.length) (0,_util_event_js__WEBPACK_IMPORTED_MODULE_3__.signal)(hidden[i], "hide")
  if (unhidden) for (let i = 0; i < unhidden.length; ++i)
    if (unhidden[i].lines.length) (0,_util_event_js__WEBPACK_IMPORTED_MODULE_3__.signal)(unhidden[i], "unhide")

  if (display.wrapper.offsetHeight)
    doc.scrollTop = cm.display.scroller.scrollTop

  // Fire change events, and delayed event handlers
  if (op.changeObjs)
    (0,_util_event_js__WEBPACK_IMPORTED_MODULE_3__.signal)(cm, "changes", cm, op.changeObjs)
  if (op.update)
    op.update.finish()
}

// Run the given function in an operation
function runInOp(cm, f) {
  if (cm.curOp) return f()
  startOperation(cm)
  try { return f() }
  finally { endOperation(cm) }
}
// Wraps a function in an operation. Returns the wrapped function.
function operation(cm, f) {
  return function() {
    if (cm.curOp) return f.apply(cm, arguments)
    startOperation(cm)
    try { return f.apply(cm, arguments) }
    finally { endOperation(cm) }
  }
}
// Used to add methods to editor and doc instances, wrapping them in
// operations.
function methodOp(f) {
  return function() {
    if (this.curOp) return f.apply(this, arguments)
    startOperation(this)
    try { return f.apply(this, arguments) }
    finally { endOperation(this) }
  }
}
function docMethodOp(f) {
  return function() {
    let cm = this.cm
    if (!cm || cm.curOp) return f.apply(this, arguments)
    startOperation(cm)
    try { return f.apply(this, arguments) }
    finally { endOperation(cm) }
  }
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/scrollbars.js":
/*!***********************************************************!*\
  !*** ./node_modules/codemirror/src/display/scrollbars.js ***!
  \***********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "measureForScrollbars": () => (/* binding */ measureForScrollbars),
/* harmony export */   "updateScrollbars": () => (/* binding */ updateScrollbars),
/* harmony export */   "scrollbarModel": () => (/* binding */ scrollbarModel),
/* harmony export */   "initScrollbars": () => (/* binding */ initScrollbars)
/* harmony export */ });
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _update_lines_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ./update_lines.js */ "./node_modules/codemirror/src/display/update_lines.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _scrolling_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ./scrolling.js */ "./node_modules/codemirror/src/display/scrolling.js");









// SCROLLBARS

// Prepare DOM reads needed to update the scrollbars. Done in one
// shot to minimize update/measure roundtrips.
function measureForScrollbars(cm) {
  let d = cm.display, gutterW = d.gutters.offsetWidth
  let docH = Math.round(cm.doc.height + (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__.paddingVert)(cm.display))
  return {
    clientHeight: d.scroller.clientHeight,
    viewHeight: d.wrapper.clientHeight,
    scrollWidth: d.scroller.scrollWidth, clientWidth: d.scroller.clientWidth,
    viewWidth: d.wrapper.clientWidth,
    barLeft: cm.options.fixedGutter ? gutterW : 0,
    docHeight: docH,
    scrollHeight: docH + (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__.scrollGap)(cm) + d.barHeight,
    nativeBarWidth: d.nativeBarWidth,
    gutterWidth: gutterW
  }
}

class NativeScrollbars {
  constructor(place, scroll, cm) {
    this.cm = cm
    let vert = this.vert = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("div", [(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("div", null, null, "min-width: 1px")], "CodeMirror-vscrollbar")
    let horiz = this.horiz = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("div", [(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("div", null, null, "height: 100%; min-height: 1px")], "CodeMirror-hscrollbar")
    vert.tabIndex = horiz.tabIndex = -1
    place(vert); place(horiz)

    ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_1__.on)(vert, "scroll", () => {
      if (vert.clientHeight) scroll(vert.scrollTop, "vertical")
    })
    ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_1__.on)(horiz, "scroll", () => {
      if (horiz.clientWidth) scroll(horiz.scrollLeft, "horizontal")
    })

    this.checkedZeroWidth = false
    // Need to set a minimum width to see the scrollbar on IE7 (but must not set it on IE8).
    if (_util_browser_js__WEBPACK_IMPORTED_MODULE_3__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_3__.ie_version < 8) this.horiz.style.minHeight = this.vert.style.minWidth = "18px"
  }

  update(measure) {
    let needsH = measure.scrollWidth > measure.clientWidth + 1
    let needsV = measure.scrollHeight > measure.clientHeight + 1
    let sWidth = measure.nativeBarWidth

    if (needsV) {
      this.vert.style.display = "block"
      this.vert.style.bottom = needsH ? sWidth + "px" : "0"
      let totalHeight = measure.viewHeight - (needsH ? sWidth : 0)
      // A bug in IE8 can cause this value to be negative, so guard it.
      this.vert.firstChild.style.height =
        Math.max(0, measure.scrollHeight - measure.clientHeight + totalHeight) + "px"
    } else {
      this.vert.style.display = ""
      this.vert.firstChild.style.height = "0"
    }

    if (needsH) {
      this.horiz.style.display = "block"
      this.horiz.style.right = needsV ? sWidth + "px" : "0"
      this.horiz.style.left = measure.barLeft + "px"
      let totalWidth = measure.viewWidth - measure.barLeft - (needsV ? sWidth : 0)
      this.horiz.firstChild.style.width =
        Math.max(0, measure.scrollWidth - measure.clientWidth + totalWidth) + "px"
    } else {
      this.horiz.style.display = ""
      this.horiz.firstChild.style.width = "0"
    }

    if (!this.checkedZeroWidth && measure.clientHeight > 0) {
      if (sWidth == 0) this.zeroWidthHack()
      this.checkedZeroWidth = true
    }

    return {right: needsV ? sWidth : 0, bottom: needsH ? sWidth : 0}
  }

  setScrollLeft(pos) {
    if (this.horiz.scrollLeft != pos) this.horiz.scrollLeft = pos
    if (this.disableHoriz) this.enableZeroWidthBar(this.horiz, this.disableHoriz, "horiz")
  }

  setScrollTop(pos) {
    if (this.vert.scrollTop != pos) this.vert.scrollTop = pos
    if (this.disableVert) this.enableZeroWidthBar(this.vert, this.disableVert, "vert")
  }

  zeroWidthHack() {
    let w = _util_browser_js__WEBPACK_IMPORTED_MODULE_3__.mac && !_util_browser_js__WEBPACK_IMPORTED_MODULE_3__.mac_geMountainLion ? "12px" : "18px"
    this.horiz.style.height = this.vert.style.width = w
    this.horiz.style.pointerEvents = this.vert.style.pointerEvents = "none"
    this.disableHoriz = new _util_misc_js__WEBPACK_IMPORTED_MODULE_5__.Delayed
    this.disableVert = new _util_misc_js__WEBPACK_IMPORTED_MODULE_5__.Delayed
  }

  enableZeroWidthBar(bar, delay, type) {
    bar.style.pointerEvents = "auto"
    function maybeDisable() {
      // To find out whether the scrollbar is still visible, we
      // check whether the element under the pixel in the bottom
      // right corner of the scrollbar box is the scrollbar box
      // itself (when the bar is still visible) or its filler child
      // (when the bar is hidden). If it is still visible, we keep
      // it enabled, if it's hidden, we disable pointer events.
      let box = bar.getBoundingClientRect()
      let elt = type == "vert" ? document.elementFromPoint(box.right - 1, (box.top + box.bottom) / 2)
          : document.elementFromPoint((box.right + box.left) / 2, box.bottom - 1)
      if (elt != bar) bar.style.pointerEvents = "none"
      else delay.set(1000, maybeDisable)
    }
    delay.set(1000, maybeDisable)
  }

  clear() {
    let parent = this.horiz.parentNode
    parent.removeChild(this.horiz)
    parent.removeChild(this.vert)
  }
}

class NullScrollbars {
  update() { return {bottom: 0, right: 0} }
  setScrollLeft() {}
  setScrollTop() {}
  clear() {}
}

function updateScrollbars(cm, measure) {
  if (!measure) measure = measureForScrollbars(cm)
  let startWidth = cm.display.barWidth, startHeight = cm.display.barHeight
  updateScrollbarsInner(cm, measure)
  for (let i = 0; i < 4 && startWidth != cm.display.barWidth || startHeight != cm.display.barHeight; i++) {
    if (startWidth != cm.display.barWidth && cm.options.lineWrapping)
      (0,_update_lines_js__WEBPACK_IMPORTED_MODULE_4__.updateHeightsInViewport)(cm)
    updateScrollbarsInner(cm, measureForScrollbars(cm))
    startWidth = cm.display.barWidth; startHeight = cm.display.barHeight
  }
}

// Re-synchronize the fake scrollbars with the actual size of the
// content.
function updateScrollbarsInner(cm, measure) {
  let d = cm.display
  let sizes = d.scrollbars.update(measure)

  d.sizer.style.paddingRight = (d.barWidth = sizes.right) + "px"
  d.sizer.style.paddingBottom = (d.barHeight = sizes.bottom) + "px"
  d.heightForcer.style.borderBottom = sizes.bottom + "px solid transparent"

  if (sizes.right && sizes.bottom) {
    d.scrollbarFiller.style.display = "block"
    d.scrollbarFiller.style.height = sizes.bottom + "px"
    d.scrollbarFiller.style.width = sizes.right + "px"
  } else d.scrollbarFiller.style.display = ""
  if (sizes.bottom && cm.options.coverGutterNextToScrollbar && cm.options.fixedGutter) {
    d.gutterFiller.style.display = "block"
    d.gutterFiller.style.height = sizes.bottom + "px"
    d.gutterFiller.style.width = measure.gutterWidth + "px"
  } else d.gutterFiller.style.display = ""
}

let scrollbarModel = {"native": NativeScrollbars, "null": NullScrollbars}

function initScrollbars(cm) {
  if (cm.display.scrollbars) {
    cm.display.scrollbars.clear()
    if (cm.display.scrollbars.addClass)
      (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_0__.rmClass)(cm.display.wrapper, cm.display.scrollbars.addClass)
  }

  cm.display.scrollbars = new scrollbarModel[cm.options.scrollbarStyle](node => {
    cm.display.wrapper.insertBefore(node, cm.display.scrollbarFiller)
    // Prevent clicks in the scrollbars from killing focus
    ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_1__.on)(node, "mousedown", () => {
      if (cm.state.focused) setTimeout(() => cm.display.input.focus(), 0)
    })
    node.setAttribute("cm-not-content", "true")
  }, (pos, axis) => {
    if (axis == "horizontal") (0,_scrolling_js__WEBPACK_IMPORTED_MODULE_6__.setScrollLeft)(cm, pos)
    else (0,_scrolling_js__WEBPACK_IMPORTED_MODULE_6__.updateScrollTop)(cm, pos)
  }, cm)
  if (cm.display.scrollbars.addClass)
    (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_0__.addClass)(cm.display.wrapper, cm.display.scrollbars.addClass)
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/scrolling.js":
/*!**********************************************************!*\
  !*** ./node_modules/codemirror/src/display/scrolling.js ***!
  \**********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "maybeScrollWindow": () => (/* binding */ maybeScrollWindow),
/* harmony export */   "scrollPosIntoView": () => (/* binding */ scrollPosIntoView),
/* harmony export */   "scrollIntoView": () => (/* binding */ scrollIntoView),
/* harmony export */   "addToScrollTop": () => (/* binding */ addToScrollTop),
/* harmony export */   "ensureCursorVisible": () => (/* binding */ ensureCursorVisible),
/* harmony export */   "scrollToCoords": () => (/* binding */ scrollToCoords),
/* harmony export */   "scrollToRange": () => (/* binding */ scrollToRange),
/* harmony export */   "scrollToCoordsRange": () => (/* binding */ scrollToCoordsRange),
/* harmony export */   "updateScrollTop": () => (/* binding */ updateScrollTop),
/* harmony export */   "setScrollTop": () => (/* binding */ setScrollTop),
/* harmony export */   "setScrollLeft": () => (/* binding */ setScrollLeft)
/* harmony export */ });
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _highlight_worker_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ./highlight_worker.js */ "./node_modules/codemirror/src/display/highlight_worker.js");
/* harmony import */ var _line_numbers_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ./line_numbers.js */ "./node_modules/codemirror/src/display/line_numbers.js");
/* harmony import */ var _update_display_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ./update_display.js */ "./node_modules/codemirror/src/display/update_display.js");










// SCROLLING THINGS INTO VIEW

// If an editor sits on the top or bottom of the window, partially
// scrolled out of view, this ensures that the cursor is visible.
function maybeScrollWindow(cm, rect) {
  if ((0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.signalDOMEvent)(cm, "scrollCursorIntoView")) return

  let display = cm.display, box = display.sizer.getBoundingClientRect(), doScroll = null
  if (rect.top + box.top < 0) doScroll = true
  else if (rect.bottom + box.top > (window.innerHeight || document.documentElement.clientHeight)) doScroll = false
  if (doScroll != null && !_util_browser_js__WEBPACK_IMPORTED_MODULE_2__.phantom) {
    let scrollNode = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.elt)("div", "\u200b", null, `position: absolute;
                         top: ${rect.top - display.viewOffset - (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.paddingTop)(cm.display)}px;
                         height: ${rect.bottom - rect.top + (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.scrollGap)(cm) + display.barHeight}px;
                         left: ${rect.left}px; width: ${Math.max(2, rect.right - rect.left)}px;`)
    cm.display.lineSpace.appendChild(scrollNode)
    scrollNode.scrollIntoView(doScroll)
    cm.display.lineSpace.removeChild(scrollNode)
  }
}

// Scroll a given position into view (immediately), verifying that
// it actually became visible (as line heights are accurately
// measured, the position of something may 'drift' during drawing).
function scrollPosIntoView(cm, pos, end, margin) {
  if (margin == null) margin = 0
  let rect
  if (!cm.options.lineWrapping && pos == end) {
    // Set pos and end to the cursor positions around the character pos sticks to
    // If pos.sticky == "before", that is around pos.ch - 1, otherwise around pos.ch
    // If pos == Pos(_, 0, "before"), pos and end are unchanged
    end = pos.sticky == "before" ? (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos)(pos.line, pos.ch + 1, "before") : pos
    pos = pos.ch ? (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos)(pos.line, pos.sticky == "before" ? pos.ch - 1 : pos.ch, "after") : pos
  }
  for (let limit = 0; limit < 5; limit++) {
    let changed = false
    let coords = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.cursorCoords)(cm, pos)
    let endCoords = !end || end == pos ? coords : (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.cursorCoords)(cm, end)
    rect = {left: Math.min(coords.left, endCoords.left),
            top: Math.min(coords.top, endCoords.top) - margin,
            right: Math.max(coords.left, endCoords.left),
            bottom: Math.max(coords.bottom, endCoords.bottom) + margin}
    let scrollPos = calculateScrollPos(cm, rect)
    let startTop = cm.doc.scrollTop, startLeft = cm.doc.scrollLeft
    if (scrollPos.scrollTop != null) {
      updateScrollTop(cm, scrollPos.scrollTop)
      if (Math.abs(cm.doc.scrollTop - startTop) > 1) changed = true
    }
    if (scrollPos.scrollLeft != null) {
      setScrollLeft(cm, scrollPos.scrollLeft)
      if (Math.abs(cm.doc.scrollLeft - startLeft) > 1) changed = true
    }
    if (!changed) break
  }
  return rect
}

// Scroll a given set of coordinates into view (immediately).
function scrollIntoView(cm, rect) {
  let scrollPos = calculateScrollPos(cm, rect)
  if (scrollPos.scrollTop != null) updateScrollTop(cm, scrollPos.scrollTop)
  if (scrollPos.scrollLeft != null) setScrollLeft(cm, scrollPos.scrollLeft)
}

// Calculate a new scroll position needed to scroll the given
// rectangle into view. Returns an object with scrollTop and
// scrollLeft properties. When these are undefined, the
// vertical/horizontal position does not need to be adjusted.
function calculateScrollPos(cm, rect) {
  let display = cm.display, snapMargin = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.textHeight)(cm.display)
  if (rect.top < 0) rect.top = 0
  let screentop = cm.curOp && cm.curOp.scrollTop != null ? cm.curOp.scrollTop : display.scroller.scrollTop
  let screen = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.displayHeight)(cm), result = {}
  if (rect.bottom - rect.top > screen) rect.bottom = rect.top + screen
  let docBottom = cm.doc.height + (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.paddingVert)(display)
  let atTop = rect.top < snapMargin, atBottom = rect.bottom > docBottom - snapMargin
  if (rect.top < screentop) {
    result.scrollTop = atTop ? 0 : rect.top
  } else if (rect.bottom > screentop + screen) {
    let newTop = Math.min(rect.top, (atBottom ? docBottom : rect.bottom) - screen)
    if (newTop != screentop) result.scrollTop = newTop
  }

  let gutterSpace = cm.options.fixedGutter ? 0 : display.gutters.offsetWidth
  let screenleft = cm.curOp && cm.curOp.scrollLeft != null ? cm.curOp.scrollLeft : display.scroller.scrollLeft - gutterSpace
  let screenw = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.displayWidth)(cm) - display.gutters.offsetWidth
  let tooWide = rect.right - rect.left > screenw
  if (tooWide) rect.right = rect.left + screenw
  if (rect.left < 10)
    result.scrollLeft = 0
  else if (rect.left < screenleft)
    result.scrollLeft = Math.max(0, rect.left + gutterSpace - (tooWide ? 0 : 10))
  else if (rect.right > screenw + screenleft - 3)
    result.scrollLeft = rect.right + (tooWide ? 0 : 10) - screenw
  return result
}

// Store a relative adjustment to the scroll position in the current
// operation (to be applied when the operation finishes).
function addToScrollTop(cm, top) {
  if (top == null) return
  resolveScrollToPos(cm)
  cm.curOp.scrollTop = (cm.curOp.scrollTop == null ? cm.doc.scrollTop : cm.curOp.scrollTop) + top
}

// Make sure that at the end of the operation the current cursor is
// shown.
function ensureCursorVisible(cm) {
  resolveScrollToPos(cm)
  let cur = cm.getCursor()
  cm.curOp.scrollToPos = {from: cur, to: cur, margin: cm.options.cursorScrollMargin}
}

function scrollToCoords(cm, x, y) {
  if (x != null || y != null) resolveScrollToPos(cm)
  if (x != null) cm.curOp.scrollLeft = x
  if (y != null) cm.curOp.scrollTop = y
}

function scrollToRange(cm, range) {
  resolveScrollToPos(cm)
  cm.curOp.scrollToPos = range
}

// When an operation has its scrollToPos property set, and another
// scroll action is applied before the end of the operation, this
// 'simulates' scrolling that position into view in a cheap way, so
// that the effect of intermediate scroll commands is not ignored.
function resolveScrollToPos(cm) {
  let range = cm.curOp.scrollToPos
  if (range) {
    cm.curOp.scrollToPos = null
    let from = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.estimateCoords)(cm, range.from), to = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.estimateCoords)(cm, range.to)
    scrollToCoordsRange(cm, from, to, range.margin)
  }
}

function scrollToCoordsRange(cm, from, to, margin) {
  let sPos = calculateScrollPos(cm, {
    left: Math.min(from.left, to.left),
    top: Math.min(from.top, to.top) - margin,
    right: Math.max(from.right, to.right),
    bottom: Math.max(from.bottom, to.bottom) + margin
  })
  scrollToCoords(cm, sPos.scrollLeft, sPos.scrollTop)
}

// Sync the scrollable area and scrollbars, ensure the viewport
// covers the visible area.
function updateScrollTop(cm, val) {
  if (Math.abs(cm.doc.scrollTop - val) < 2) return
  if (!_util_browser_js__WEBPACK_IMPORTED_MODULE_2__.gecko) (0,_update_display_js__WEBPACK_IMPORTED_MODULE_7__.updateDisplaySimple)(cm, {top: val})
  setScrollTop(cm, val, true)
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_2__.gecko) (0,_update_display_js__WEBPACK_IMPORTED_MODULE_7__.updateDisplaySimple)(cm)
  ;(0,_highlight_worker_js__WEBPACK_IMPORTED_MODULE_5__.startWorker)(cm, 100)
}

function setScrollTop(cm, val, forceScroll) {
  val = Math.max(0, Math.min(cm.display.scroller.scrollHeight - cm.display.scroller.clientHeight, val))
  if (cm.display.scroller.scrollTop == val && !forceScroll) return
  cm.doc.scrollTop = val
  cm.display.scrollbars.setScrollTop(val)
  if (cm.display.scroller.scrollTop != val) cm.display.scroller.scrollTop = val
}

// Sync scroller and scrollbar, ensure the gutter elements are
// aligned.
function setScrollLeft(cm, val, isScroller, forceScroll) {
  val = Math.max(0, Math.min(val, cm.display.scroller.scrollWidth - cm.display.scroller.clientWidth))
  if ((isScroller ? val == cm.doc.scrollLeft : Math.abs(cm.doc.scrollLeft - val) < 2) && !forceScroll) return
  cm.doc.scrollLeft = val
  ;(0,_line_numbers_js__WEBPACK_IMPORTED_MODULE_6__.alignHorizontally)(cm)
  if (cm.display.scroller.scrollLeft != val) cm.display.scroller.scrollLeft = val
  cm.display.scrollbars.setScrollLeft(val)
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/selection.js":
/*!**********************************************************!*\
  !*** ./node_modules/codemirror/src/display/selection.js ***!
  \**********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "updateSelection": () => (/* binding */ updateSelection),
/* harmony export */   "prepareSelection": () => (/* binding */ prepareSelection),
/* harmony export */   "drawSelectionCursor": () => (/* binding */ drawSelectionCursor),
/* harmony export */   "restartBlink": () => (/* binding */ restartBlink)
/* harmony export */ });
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_bidi_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/bidi.js */ "./node_modules/codemirror/src/util/bidi.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _focus_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ./focus.js */ "./node_modules/codemirror/src/display/focus.js");








function updateSelection(cm) {
  cm.display.input.showSelection(cm.display.input.prepareSelection())
}

function prepareSelection(cm, primary = true) {
  let doc = cm.doc, result = {}
  let curFragment = result.cursors = document.createDocumentFragment()
  let selFragment = result.selection = document.createDocumentFragment()

  for (let i = 0; i < doc.sel.ranges.length; i++) {
    if (!primary && i == doc.sel.primIndex) continue
    let range = doc.sel.ranges[i]
    if (range.from().line >= cm.display.viewTo || range.to().line < cm.display.viewFrom) continue
    let collapsed = range.empty()
    if (collapsed || cm.options.showCursorWhenSelecting)
      drawSelectionCursor(cm, range.head, curFragment)
    if (!collapsed)
      drawSelectionRange(cm, range, selFragment)
  }
  return result
}

// Draws a cursor for the given range
function drawSelectionCursor(cm, head, output) {
  let pos = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.cursorCoords)(cm, head, "div", null, null, !cm.options.singleCursorHeightPerLine)

  let cursor = output.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.elt)("div", "\u00a0", "CodeMirror-cursor"))
  cursor.style.left = pos.left + "px"
  cursor.style.top = pos.top + "px"
  cursor.style.height = Math.max(0, pos.bottom - pos.top) * cm.options.cursorHeight + "px"

  if (/\bcm-fat-cursor\b/.test(cm.getWrapperElement().className)) {
    let charPos = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.charCoords)(cm, head, "div", null, null)
    cursor.style.width = Math.max(0, charPos.right - charPos.left) + "px"
  }

  if (pos.other) {
    // Secondary cursor, shown when on a 'jump' in bi-directional text
    let otherCursor = output.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.elt)("div", "\u00a0", "CodeMirror-cursor CodeMirror-secondarycursor"))
    otherCursor.style.display = ""
    otherCursor.style.left = pos.other.left + "px"
    otherCursor.style.top = pos.other.top + "px"
    otherCursor.style.height = (pos.other.bottom - pos.other.top) * .85 + "px"
  }
}

function cmpCoords(a, b) { return a.top - b.top || a.left - b.left }

// Draws the given range as a highlighted selection
function drawSelectionRange(cm, range, output) {
  let display = cm.display, doc = cm.doc
  let fragment = document.createDocumentFragment()
  let padding = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.paddingH)(cm.display), leftSide = padding.left
  let rightSide = Math.max(display.sizerWidth, (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.displayWidth)(cm) - display.sizer.offsetLeft) - padding.right
  let docLTR = doc.direction == "ltr"

  function add(left, top, width, bottom) {
    if (top < 0) top = 0
    top = Math.round(top)
    bottom = Math.round(bottom)
    fragment.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.elt)("div", null, "CodeMirror-selected", `position: absolute; left: ${left}px;
                             top: ${top}px; width: ${width == null ? rightSide - left : width}px;
                             height: ${bottom - top}px`))
  }

  function drawForLine(line, fromArg, toArg) {
    let lineObj = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__.getLine)(doc, line)
    let lineLen = lineObj.text.length
    let start, end
    function coords(ch, bias) {
      return (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.charCoords)(cm, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos)(line, ch), "div", lineObj, bias)
    }

    function wrapX(pos, dir, side) {
      let extent = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.wrappedLineExtentChar)(cm, lineObj, null, pos)
      let prop = (dir == "ltr") == (side == "after") ? "left" : "right"
      let ch = side == "after" ? extent.begin : extent.end - (/\s/.test(lineObj.text.charAt(extent.end - 1)) ? 2 : 1)
      return coords(ch, prop)[prop]
    }

    let order = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_4__.getOrder)(lineObj, doc.direction)
    ;(0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_4__.iterateBidiSections)(order, fromArg || 0, toArg == null ? lineLen : toArg, (from, to, dir, i) => {
      let ltr = dir == "ltr"
      let fromPos = coords(from, ltr ? "left" : "right")
      let toPos = coords(to - 1, ltr ? "right" : "left")

      let openStart = fromArg == null && from == 0, openEnd = toArg == null && to == lineLen
      let first = i == 0, last = !order || i == order.length - 1
      if (toPos.top - fromPos.top <= 3) { // Single line
        let openLeft = (docLTR ? openStart : openEnd) && first
        let openRight = (docLTR ? openEnd : openStart) && last
        let left = openLeft ? leftSide : (ltr ? fromPos : toPos).left
        let right = openRight ? rightSide : (ltr ? toPos : fromPos).right
        add(left, fromPos.top, right - left, fromPos.bottom)
      } else { // Multiple lines
        let topLeft, topRight, botLeft, botRight
        if (ltr) {
          topLeft = docLTR && openStart && first ? leftSide : fromPos.left
          topRight = docLTR ? rightSide : wrapX(from, dir, "before")
          botLeft = docLTR ? leftSide : wrapX(to, dir, "after")
          botRight = docLTR && openEnd && last ? rightSide : toPos.right
        } else {
          topLeft = !docLTR ? leftSide : wrapX(from, dir, "before")
          topRight = !docLTR && openStart && first ? rightSide : fromPos.right
          botLeft = !docLTR && openEnd && last ? leftSide : toPos.left
          botRight = !docLTR ? rightSide : wrapX(to, dir, "after")
        }
        add(topLeft, fromPos.top, topRight - topLeft, fromPos.bottom)
        if (fromPos.bottom < toPos.top) add(leftSide, fromPos.bottom, null, toPos.top)
        add(botLeft, toPos.top, botRight - botLeft, toPos.bottom)
      }

      if (!start || cmpCoords(fromPos, start) < 0) start = fromPos
      if (cmpCoords(toPos, start) < 0) start = toPos
      if (!end || cmpCoords(fromPos, end) < 0) end = fromPos
      if (cmpCoords(toPos, end) < 0) end = toPos
    })
    return {start: start, end: end}
  }

  let sFrom = range.from(), sTo = range.to()
  if (sFrom.line == sTo.line) {
    drawForLine(sFrom.line, sFrom.ch, sTo.ch)
  } else {
    let fromLine = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__.getLine)(doc, sFrom.line), toLine = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__.getLine)(doc, sTo.line)
    let singleVLine = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_1__.visualLine)(fromLine) == (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_1__.visualLine)(toLine)
    let leftEnd = drawForLine(sFrom.line, sFrom.ch, singleVLine ? fromLine.text.length + 1 : null).end
    let rightStart = drawForLine(sTo.line, singleVLine ? 0 : null, sTo.ch).start
    if (singleVLine) {
      if (leftEnd.top < rightStart.top - 2) {
        add(leftEnd.right, leftEnd.top, null, leftEnd.bottom)
        add(leftSide, rightStart.top, rightStart.left, rightStart.bottom)
      } else {
        add(leftEnd.right, leftEnd.top, rightStart.left - leftEnd.right, leftEnd.bottom)
      }
    }
    if (leftEnd.bottom < rightStart.top)
      add(leftSide, leftEnd.bottom, null, rightStart.top)
  }

  output.appendChild(fragment)
}

// Cursor-blinking
function restartBlink(cm) {
  if (!cm.state.focused) return
  let display = cm.display
  clearInterval(display.blinker)
  let on = true
  display.cursorDiv.style.visibility = ""
  if (cm.options.cursorBlinkRate > 0)
    display.blinker = setInterval(() => {
      if (!cm.hasFocus()) (0,_focus_js__WEBPACK_IMPORTED_MODULE_6__.onBlur)(cm)
      display.cursorDiv.style.visibility = (on = !on) ? "" : "hidden"
    }, cm.options.cursorBlinkRate)
  else if (cm.options.cursorBlinkRate < 0)
    display.cursorDiv.style.visibility = "hidden"
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/update_display.js":
/*!***************************************************************!*\
  !*** ./node_modules/codemirror/src/display/update_display.js ***!
  \***************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "DisplayUpdate": () => (/* binding */ DisplayUpdate),
/* harmony export */   "maybeClipScrollbars": () => (/* binding */ maybeClipScrollbars),
/* harmony export */   "updateDisplayIfNeeded": () => (/* binding */ updateDisplayIfNeeded),
/* harmony export */   "postUpdateDisplay": () => (/* binding */ postUpdateDisplay),
/* harmony export */   "updateDisplaySimple": () => (/* binding */ updateDisplaySimple),
/* harmony export */   "updateGutterSpace": () => (/* binding */ updateGutterSpace),
/* harmony export */   "setDocumentHeight": () => (/* binding */ setDocumentHeight)
/* harmony export */ });
/* harmony import */ var _line_saw_special_spans_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/saw_special_spans.js */ "./node_modules/codemirror/src/line/saw_special_spans.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _util_operation_group_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../util/operation_group.js */ "./node_modules/codemirror/src/util/operation_group.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _update_line_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ./update_line.js */ "./node_modules/codemirror/src/display/update_line.js");
/* harmony import */ var _highlight_worker_js__WEBPACK_IMPORTED_MODULE_10__ = __webpack_require__(/*! ./highlight_worker.js */ "./node_modules/codemirror/src/display/highlight_worker.js");
/* harmony import */ var _line_numbers_js__WEBPACK_IMPORTED_MODULE_11__ = __webpack_require__(/*! ./line_numbers.js */ "./node_modules/codemirror/src/display/line_numbers.js");
/* harmony import */ var _scrollbars_js__WEBPACK_IMPORTED_MODULE_12__ = __webpack_require__(/*! ./scrollbars.js */ "./node_modules/codemirror/src/display/scrollbars.js");
/* harmony import */ var _selection_js__WEBPACK_IMPORTED_MODULE_13__ = __webpack_require__(/*! ./selection.js */ "./node_modules/codemirror/src/display/selection.js");
/* harmony import */ var _update_lines_js__WEBPACK_IMPORTED_MODULE_14__ = __webpack_require__(/*! ./update_lines.js */ "./node_modules/codemirror/src/display/update_lines.js");
/* harmony import */ var _view_tracking_js__WEBPACK_IMPORTED_MODULE_15__ = __webpack_require__(/*! ./view_tracking.js */ "./node_modules/codemirror/src/display/view_tracking.js");


















// DISPLAY DRAWING

class DisplayUpdate {
  constructor(cm, viewport, force) {
    let display = cm.display

    this.viewport = viewport
    // Store some values that we'll need later (but don't want to force a relayout for)
    this.visible = (0,_update_lines_js__WEBPACK_IMPORTED_MODULE_14__.visibleLines)(display, cm.doc, viewport)
    this.editorIsHidden = !display.wrapper.offsetWidth
    this.wrapperHeight = display.wrapper.clientHeight
    this.wrapperWidth = display.wrapper.clientWidth
    this.oldDisplayWidth = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.displayWidth)(cm)
    this.force = force
    this.dims = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.getDimensions)(cm)
    this.events = []
  }

  signal(emitter, type) {
    if ((0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.hasHandler)(emitter, type))
      this.events.push(arguments)
  }
  finish() {
    for (let i = 0; i < this.events.length; i++)
      _util_event_js__WEBPACK_IMPORTED_MODULE_6__.signal.apply(null, this.events[i])
  }
}

function maybeClipScrollbars(cm) {
  let display = cm.display
  if (!display.scrollbarsClipped && display.scroller.offsetWidth) {
    display.nativeBarWidth = display.scroller.offsetWidth - display.scroller.clientWidth
    display.heightForcer.style.height = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.scrollGap)(cm) + "px"
    display.sizer.style.marginBottom = -display.nativeBarWidth + "px"
    display.sizer.style.borderRightWidth = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.scrollGap)(cm) + "px"
    display.scrollbarsClipped = true
  }
}

function selectionSnapshot(cm) {
  if (cm.hasFocus()) return null
  let active = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.activeElt)()
  if (!active || !(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.contains)(cm.display.lineDiv, active)) return null
  let result = {activeElt: active}
  if (window.getSelection) {
    let sel = window.getSelection()
    if (sel.anchorNode && sel.extend && (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.contains)(cm.display.lineDiv, sel.anchorNode)) {
      result.anchorNode = sel.anchorNode
      result.anchorOffset = sel.anchorOffset
      result.focusNode = sel.focusNode
      result.focusOffset = sel.focusOffset
    }
  }
  return result
}

function restoreSelection(snapshot) {
  if (!snapshot || !snapshot.activeElt || snapshot.activeElt == (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.activeElt)()) return
  snapshot.activeElt.focus()
  if (!/^(INPUT|TEXTAREA)$/.test(snapshot.activeElt.nodeName) &&
      snapshot.anchorNode && (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.contains)(document.body, snapshot.anchorNode) && (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.contains)(document.body, snapshot.focusNode)) {
    let sel = window.getSelection(), range = document.createRange()
    range.setEnd(snapshot.anchorNode, snapshot.anchorOffset)
    range.collapse(false)
    sel.removeAllRanges()
    sel.addRange(range)
    sel.extend(snapshot.focusNode, snapshot.focusOffset)
  }
}

// Does the actual updating of the line display. Bails out
// (returning false) when there is nothing to be done and forced is
// false.
function updateDisplayIfNeeded(cm, update) {
  let display = cm.display, doc = cm.doc

  if (update.editorIsHidden) {
    (0,_view_tracking_js__WEBPACK_IMPORTED_MODULE_15__.resetView)(cm)
    return false
  }

  // Bail out if the visible area is already rendered and nothing changed.
  if (!update.force &&
      update.visible.from >= display.viewFrom && update.visible.to <= display.viewTo &&
      (display.updateLineNumbers == null || display.updateLineNumbers >= display.viewTo) &&
      display.renderedView == display.view && (0,_view_tracking_js__WEBPACK_IMPORTED_MODULE_15__.countDirtyView)(cm) == 0)
    return false

  if ((0,_line_numbers_js__WEBPACK_IMPORTED_MODULE_11__.maybeUpdateLineNumberWidth)(cm)) {
    (0,_view_tracking_js__WEBPACK_IMPORTED_MODULE_15__.resetView)(cm)
    update.dims = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.getDimensions)(cm)
  }

  // Compute a suitable new viewport (from & to)
  let end = doc.first + doc.size
  let from = Math.max(update.visible.from - cm.options.viewportMargin, doc.first)
  let to = Math.min(end, update.visible.to + cm.options.viewportMargin)
  if (display.viewFrom < from && from - display.viewFrom < 20) from = Math.max(doc.first, display.viewFrom)
  if (display.viewTo > to && display.viewTo - to < 20) to = Math.min(end, display.viewTo)
  if (_line_saw_special_spans_js__WEBPACK_IMPORTED_MODULE_0__.sawCollapsedSpans) {
    from = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_1__.visualLineNo)(cm.doc, from)
    to = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_1__.visualLineEndNo)(cm.doc, to)
  }

  let different = from != display.viewFrom || to != display.viewTo ||
    display.lastWrapHeight != update.wrapperHeight || display.lastWrapWidth != update.wrapperWidth
  ;(0,_view_tracking_js__WEBPACK_IMPORTED_MODULE_15__.adjustView)(cm, from, to)

  display.viewOffset = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_1__.heightAtLine)((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__.getLine)(cm.doc, display.viewFrom))
  // Position the mover div to align with the current scroll position
  cm.display.mover.style.top = display.viewOffset + "px"

  let toUpdate = (0,_view_tracking_js__WEBPACK_IMPORTED_MODULE_15__.countDirtyView)(cm)
  if (!different && toUpdate == 0 && !update.force && display.renderedView == display.view &&
      (display.updateLineNumbers == null || display.updateLineNumbers >= display.viewTo))
    return false

  // For big changes, we hide the enclosing element during the
  // update, since that speeds up the operations on most browsers.
  let selSnapshot = selectionSnapshot(cm)
  if (toUpdate > 4) display.lineDiv.style.display = "none"
  patchDisplay(cm, display.updateLineNumbers, update.dims)
  if (toUpdate > 4) display.lineDiv.style.display = ""
  display.renderedView = display.view
  // There might have been a widget with a focused element that got
  // hidden or updated, if so re-focus it.
  restoreSelection(selSnapshot)

  // Prevent selection and cursors from interfering with the scroll
  // width and height.
  ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.removeChildren)(display.cursorDiv)
  ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.removeChildren)(display.selectionDiv)
  display.gutters.style.height = display.sizer.style.minHeight = 0

  if (different) {
    display.lastWrapHeight = update.wrapperHeight
    display.lastWrapWidth = update.wrapperWidth
    ;(0,_highlight_worker_js__WEBPACK_IMPORTED_MODULE_10__.startWorker)(cm, 400)
  }

  display.updateLineNumbers = null

  return true
}

function postUpdateDisplay(cm, update) {
  let viewport = update.viewport

  for (let first = true;; first = false) {
    if (!first || !cm.options.lineWrapping || update.oldDisplayWidth == (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.displayWidth)(cm)) {
      // Clip forced viewport to actual scrollable area.
      if (viewport && viewport.top != null)
        viewport = {top: Math.min(cm.doc.height + (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.paddingVert)(cm.display) - (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.displayHeight)(cm), viewport.top)}
      // Updated line heights might result in the drawn area not
      // actually covering the viewport. Keep looping until it does.
      update.visible = (0,_update_lines_js__WEBPACK_IMPORTED_MODULE_14__.visibleLines)(cm.display, cm.doc, viewport)
      if (update.visible.from >= cm.display.viewFrom && update.visible.to <= cm.display.viewTo)
        break
    } else if (first) {
      update.visible = (0,_update_lines_js__WEBPACK_IMPORTED_MODULE_14__.visibleLines)(cm.display, cm.doc, viewport)
    }
    if (!updateDisplayIfNeeded(cm, update)) break
    ;(0,_update_lines_js__WEBPACK_IMPORTED_MODULE_14__.updateHeightsInViewport)(cm)
    let barMeasure = (0,_scrollbars_js__WEBPACK_IMPORTED_MODULE_12__.measureForScrollbars)(cm)
    ;(0,_selection_js__WEBPACK_IMPORTED_MODULE_13__.updateSelection)(cm)
    ;(0,_scrollbars_js__WEBPACK_IMPORTED_MODULE_12__.updateScrollbars)(cm, barMeasure)
    setDocumentHeight(cm, barMeasure)
    update.force = false
  }

  update.signal(cm, "update", cm)
  if (cm.display.viewFrom != cm.display.reportedViewFrom || cm.display.viewTo != cm.display.reportedViewTo) {
    update.signal(cm, "viewportChange", cm, cm.display.viewFrom, cm.display.viewTo)
    cm.display.reportedViewFrom = cm.display.viewFrom; cm.display.reportedViewTo = cm.display.viewTo
  }
}

function updateDisplaySimple(cm, viewport) {
  let update = new DisplayUpdate(cm, viewport)
  if (updateDisplayIfNeeded(cm, update)) {
    (0,_update_lines_js__WEBPACK_IMPORTED_MODULE_14__.updateHeightsInViewport)(cm)
    postUpdateDisplay(cm, update)
    let barMeasure = (0,_scrollbars_js__WEBPACK_IMPORTED_MODULE_12__.measureForScrollbars)(cm)
    ;(0,_selection_js__WEBPACK_IMPORTED_MODULE_13__.updateSelection)(cm)
    ;(0,_scrollbars_js__WEBPACK_IMPORTED_MODULE_12__.updateScrollbars)(cm, barMeasure)
    setDocumentHeight(cm, barMeasure)
    update.finish()
  }
}

// Sync the actual display DOM structure with display.view, removing
// nodes for lines that are no longer in view, and creating the ones
// that are not there yet, and updating the ones that are out of
// date.
function patchDisplay(cm, updateNumbersFrom, dims) {
  let display = cm.display, lineNumbers = cm.options.lineNumbers
  let container = display.lineDiv, cur = container.firstChild

  function rm(node) {
    let next = node.nextSibling
    // Works around a throw-scroll bug in OS X Webkit
    if (_util_browser_js__WEBPACK_IMPORTED_MODULE_4__.webkit && _util_browser_js__WEBPACK_IMPORTED_MODULE_4__.mac && cm.display.currentWheelTarget == node)
      node.style.display = "none"
    else
      node.parentNode.removeChild(node)
    return next
  }

  let view = display.view, lineN = display.viewFrom
  // Loop over the elements in the view, syncing cur (the DOM nodes
  // in display.lineDiv) with the view as we go.
  for (let i = 0; i < view.length; i++) {
    let lineView = view[i]
    if (lineView.hidden) {
    } else if (!lineView.node || lineView.node.parentNode != container) { // Not drawn yet
      let node = (0,_update_line_js__WEBPACK_IMPORTED_MODULE_9__.buildLineElement)(cm, lineView, lineN, dims)
      container.insertBefore(node, cur)
    } else { // Already drawn
      while (cur != lineView.node) cur = rm(cur)
      let updateNumber = lineNumbers && updateNumbersFrom != null &&
        updateNumbersFrom <= lineN && lineView.lineNumber
      if (lineView.changes) {
        if ((0,_util_misc_js__WEBPACK_IMPORTED_MODULE_8__.indexOf)(lineView.changes, "gutter") > -1) updateNumber = false
        ;(0,_update_line_js__WEBPACK_IMPORTED_MODULE_9__.updateLineForChanges)(cm, lineView, lineN, dims)
      }
      if (updateNumber) {
        (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.removeChildren)(lineView.lineNumber)
        lineView.lineNumber.appendChild(document.createTextNode((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__.lineNumberFor)(cm.options, lineN)))
      }
      cur = lineView.node.nextSibling
    }
    lineN += lineView.size
  }
  while (cur) cur = rm(cur)
}

function updateGutterSpace(display) {
  let width = display.gutters.offsetWidth
  display.sizer.style.marginLeft = width + "px"
  // Send an event to consumers responding to changes in gutter width.
  ;(0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_7__.signalLater)(display, "gutterChanged", display)
}

function setDocumentHeight(cm, measure) {
  cm.display.sizer.style.minHeight = measure.docHeight + "px"
  cm.display.heightForcer.style.top = measure.docHeight + "px"
  cm.display.gutters.style.height = (measure.docHeight + cm.display.barHeight + (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.scrollGap)(cm)) + "px"
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/update_line.js":
/*!************************************************************!*\
  !*** ./node_modules/codemirror/src/display/update_line.js ***!
  \************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "updateLineForChanges": () => (/* binding */ updateLineForChanges),
/* harmony export */   "buildLineElement": () => (/* binding */ buildLineElement)
/* harmony export */ });
/* harmony import */ var _line_line_data_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/line_data.js */ "./node_modules/codemirror/src/line/line_data.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_operation_group_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/operation_group.js */ "./node_modules/codemirror/src/util/operation_group.js");






// When an aspect of a line changes, a string is added to
// lineView.changes. This updates the relevant part of the line's
// DOM structure.
function updateLineForChanges(cm, lineView, lineN, dims) {
  for (let j = 0; j < lineView.changes.length; j++) {
    let type = lineView.changes[j]
    if (type == "text") updateLineText(cm, lineView)
    else if (type == "gutter") updateLineGutter(cm, lineView, lineN, dims)
    else if (type == "class") updateLineClasses(cm, lineView)
    else if (type == "widget") updateLineWidgets(cm, lineView, dims)
  }
  lineView.changes = null
}

// Lines with gutter elements, widgets or a background class need to
// be wrapped, and have the extra elements added to the wrapper div
function ensureLineWrapped(lineView) {
  if (lineView.node == lineView.text) {
    lineView.node = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.elt)("div", null, null, "position: relative")
    if (lineView.text.parentNode)
      lineView.text.parentNode.replaceChild(lineView.node, lineView.text)
    lineView.node.appendChild(lineView.text)
    if (_util_browser_js__WEBPACK_IMPORTED_MODULE_2__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_2__.ie_version < 8) lineView.node.style.zIndex = 2
  }
  return lineView.node
}

function updateLineBackground(cm, lineView) {
  let cls = lineView.bgClass ? lineView.bgClass + " " + (lineView.line.bgClass || "") : lineView.line.bgClass
  if (cls) cls += " CodeMirror-linebackground"
  if (lineView.background) {
    if (cls) lineView.background.className = cls
    else { lineView.background.parentNode.removeChild(lineView.background); lineView.background = null }
  } else if (cls) {
    let wrap = ensureLineWrapped(lineView)
    lineView.background = wrap.insertBefore((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.elt)("div", null, cls), wrap.firstChild)
    cm.display.input.setUneditable(lineView.background)
  }
}

// Wrapper around buildLineContent which will reuse the structure
// in display.externalMeasured when possible.
function getLineContent(cm, lineView) {
  let ext = cm.display.externalMeasured
  if (ext && ext.line == lineView.line) {
    cm.display.externalMeasured = null
    lineView.measure = ext.measure
    return ext.built
  }
  return (0,_line_line_data_js__WEBPACK_IMPORTED_MODULE_0__.buildLineContent)(cm, lineView)
}

// Redraw the line's text. Interacts with the background and text
// classes because the mode may output tokens that influence these
// classes.
function updateLineText(cm, lineView) {
  let cls = lineView.text.className
  let built = getLineContent(cm, lineView)
  if (lineView.text == lineView.node) lineView.node = built.pre
  lineView.text.parentNode.replaceChild(built.pre, lineView.text)
  lineView.text = built.pre
  if (built.bgClass != lineView.bgClass || built.textClass != lineView.textClass) {
    lineView.bgClass = built.bgClass
    lineView.textClass = built.textClass
    updateLineClasses(cm, lineView)
  } else if (cls) {
    lineView.text.className = cls
  }
}

function updateLineClasses(cm, lineView) {
  updateLineBackground(cm, lineView)
  if (lineView.line.wrapClass)
    ensureLineWrapped(lineView).className = lineView.line.wrapClass
  else if (lineView.node != lineView.text)
    lineView.node.className = ""
  let textClass = lineView.textClass ? lineView.textClass + " " + (lineView.line.textClass || "") : lineView.line.textClass
  lineView.text.className = textClass || ""
}

function updateLineGutter(cm, lineView, lineN, dims) {
  if (lineView.gutter) {
    lineView.node.removeChild(lineView.gutter)
    lineView.gutter = null
  }
  if (lineView.gutterBackground) {
    lineView.node.removeChild(lineView.gutterBackground)
    lineView.gutterBackground = null
  }
  if (lineView.line.gutterClass) {
    let wrap = ensureLineWrapped(lineView)
    lineView.gutterBackground = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.elt)("div", null, "CodeMirror-gutter-background " + lineView.line.gutterClass,
                                    `left: ${cm.options.fixedGutter ? dims.fixedPos : -dims.gutterTotalWidth}px; width: ${dims.gutterTotalWidth}px`)
    cm.display.input.setUneditable(lineView.gutterBackground)
    wrap.insertBefore(lineView.gutterBackground, lineView.text)
  }
  let markers = lineView.line.gutterMarkers
  if (cm.options.lineNumbers || markers) {
    let wrap = ensureLineWrapped(lineView)
    let gutterWrap = lineView.gutter = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.elt)("div", null, "CodeMirror-gutter-wrapper", `left: ${cm.options.fixedGutter ? dims.fixedPos : -dims.gutterTotalWidth}px`)
    gutterWrap.setAttribute("aria-hidden", "true")
    cm.display.input.setUneditable(gutterWrap)
    wrap.insertBefore(gutterWrap, lineView.text)
    if (lineView.line.gutterClass)
      gutterWrap.className += " " + lineView.line.gutterClass
    if (cm.options.lineNumbers && (!markers || !markers["CodeMirror-linenumbers"]))
      lineView.lineNumber = gutterWrap.appendChild(
        (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.elt)("div", (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__.lineNumberFor)(cm.options, lineN),
            "CodeMirror-linenumber CodeMirror-gutter-elt",
            `left: ${dims.gutterLeft["CodeMirror-linenumbers"]}px; width: ${cm.display.lineNumInnerWidth}px`))
    if (markers) for (let k = 0; k < cm.display.gutterSpecs.length; ++k) {
      let id = cm.display.gutterSpecs[k].className, found = markers.hasOwnProperty(id) && markers[id]
      if (found)
        gutterWrap.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.elt)("div", [found], "CodeMirror-gutter-elt",
                                   `left: ${dims.gutterLeft[id]}px; width: ${dims.gutterWidth[id]}px`))
    }
  }
}

function updateLineWidgets(cm, lineView, dims) {
  if (lineView.alignable) lineView.alignable = null
  let isWidget = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.classTest)("CodeMirror-linewidget")
  for (let node = lineView.node.firstChild, next; node; node = next) {
    next = node.nextSibling
    if (isWidget.test(node.className)) lineView.node.removeChild(node)
  }
  insertLineWidgets(cm, lineView, dims)
}

// Build a line's DOM representation from scratch
function buildLineElement(cm, lineView, lineN, dims) {
  let built = getLineContent(cm, lineView)
  lineView.text = lineView.node = built.pre
  if (built.bgClass) lineView.bgClass = built.bgClass
  if (built.textClass) lineView.textClass = built.textClass

  updateLineClasses(cm, lineView)
  updateLineGutter(cm, lineView, lineN, dims)
  insertLineWidgets(cm, lineView, dims)
  return lineView.node
}

// A lineView may contain multiple logical lines (when merged by
// collapsed spans). The widgets for all of them need to be drawn.
function insertLineWidgets(cm, lineView, dims) {
  insertLineWidgetsFor(cm, lineView.line, lineView, dims, true)
  if (lineView.rest) for (let i = 0; i < lineView.rest.length; i++)
    insertLineWidgetsFor(cm, lineView.rest[i], lineView, dims, false)
}

function insertLineWidgetsFor(cm, line, lineView, dims, allowAbove) {
  if (!line.widgets) return
  let wrap = ensureLineWrapped(lineView)
  for (let i = 0, ws = line.widgets; i < ws.length; ++i) {
    let widget = ws[i], node = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.elt)("div", [widget.node], "CodeMirror-linewidget" + (widget.className ? " " + widget.className : ""))
    if (!widget.handleMouseEvents) node.setAttribute("cm-ignore-events", "true")
    positionLineWidget(widget, node, lineView, dims)
    cm.display.input.setUneditable(node)
    if (allowAbove && widget.above)
      wrap.insertBefore(node, lineView.gutter || lineView.text)
    else
      wrap.appendChild(node)
    ;(0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_4__.signalLater)(widget, "redraw")
  }
}

function positionLineWidget(widget, node, lineView, dims) {
  if (widget.noHScroll) {
    ;(lineView.alignable || (lineView.alignable = [])).push(node)
    let width = dims.wrapperWidth
    node.style.left = dims.fixedPos + "px"
    if (!widget.coverGutter) {
      width -= dims.gutterTotalWidth
      node.style.paddingLeft = dims.gutterTotalWidth + "px"
    }
    node.style.width = width + "px"
  }
  if (widget.coverGutter) {
    node.style.zIndex = 5
    node.style.position = "relative"
    if (!widget.noHScroll) node.style.marginLeft = -dims.gutterTotalWidth + "px"
  }
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/update_lines.js":
/*!*************************************************************!*\
  !*** ./node_modules/codemirror/src/display/update_lines.js ***!
  \*************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "updateHeightsInViewport": () => (/* binding */ updateHeightsInViewport),
/* harmony export */   "visibleLines": () => (/* binding */ visibleLines)
/* harmony export */ });
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");





// Read the actual heights of the rendered lines, and update their
// stored heights to match.
function updateHeightsInViewport(cm) {
  let display = cm.display
  let prevBottom = display.lineDiv.offsetTop
  for (let i = 0; i < display.view.length; i++) {
    let cur = display.view[i], wrapping = cm.options.lineWrapping
    let height, width = 0
    if (cur.hidden) continue
    if (_util_browser_js__WEBPACK_IMPORTED_MODULE_3__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_3__.ie_version < 8) {
      let bot = cur.node.offsetTop + cur.node.offsetHeight
      height = bot - prevBottom
      prevBottom = bot
    } else {
      let box = cur.node.getBoundingClientRect()
      height = box.bottom - box.top
      // Check that lines don't extend past the right of the current
      // editor width
      if (!wrapping && cur.text.firstChild)
        width = cur.text.firstChild.getBoundingClientRect().right - box.left - 1
    }
    let diff = cur.line.height - height
    if (diff > .005 || diff < -.005) {
      (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__.updateLineHeight)(cur.line, height)
      updateWidgetHeight(cur.line)
      if (cur.rest) for (let j = 0; j < cur.rest.length; j++)
        updateWidgetHeight(cur.rest[j])
    }
    if (width > cm.display.sizerWidth) {
      let chWidth = Math.ceil(width / (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__.charWidth)(cm.display))
      if (chWidth > cm.display.maxLineLength) {
        cm.display.maxLineLength = chWidth
        cm.display.maxLine = cur.line
        cm.display.maxLineChanged = true
      }
    }
  }
}

// Read and store the height of line widgets associated with the
// given line.
function updateWidgetHeight(line) {
  if (line.widgets) for (let i = 0; i < line.widgets.length; ++i) {
    let w = line.widgets[i], parent = w.node.parentNode
    if (parent) w.height = parent.offsetHeight
  }
}

// Compute the lines that are visible in a given viewport (defaults
// the the current scroll position). viewport may contain top,
// height, and ensure (see op.scrollToPos) properties.
function visibleLines(display, doc, viewport) {
  let top = viewport && viewport.top != null ? Math.max(0, viewport.top) : display.scroller.scrollTop
  top = Math.floor(top - (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_2__.paddingTop)(display))
  let bottom = viewport && viewport.bottom != null ? viewport.bottom : top + display.wrapper.clientHeight

  let from = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__.lineAtHeight)(doc, top), to = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__.lineAtHeight)(doc, bottom)
  // Ensure is a {from: {line, ch}, to: {line, ch}} object, and
  // forces those lines into the viewport (if possible).
  if (viewport && viewport.ensure) {
    let ensureFrom = viewport.ensure.from.line, ensureTo = viewport.ensure.to.line
    if (ensureFrom < from) {
      from = ensureFrom
      to = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__.lineAtHeight)(doc, (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_0__.heightAtLine)((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__.getLine)(doc, ensureFrom)) + display.wrapper.clientHeight)
    } else if (Math.min(ensureTo, doc.lastLine()) >= to) {
      from = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__.lineAtHeight)(doc, (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_0__.heightAtLine)((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_1__.getLine)(doc, ensureTo)) - display.wrapper.clientHeight)
      to = ensureTo
    }
  }
  return {from: from, to: Math.max(to, from + 1)}
}


/***/ }),

/***/ "./node_modules/codemirror/src/display/view_tracking.js":
/*!**************************************************************!*\
  !*** ./node_modules/codemirror/src/display/view_tracking.js ***!
  \**************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "regChange": () => (/* binding */ regChange),
/* harmony export */   "regLineChange": () => (/* binding */ regLineChange),
/* harmony export */   "resetView": () => (/* binding */ resetView),
/* harmony export */   "adjustView": () => (/* binding */ adjustView),
/* harmony export */   "countDirtyView": () => (/* binding */ countDirtyView)
/* harmony export */ });
/* harmony import */ var _line_line_data_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/line_data.js */ "./node_modules/codemirror/src/line/line_data.js");
/* harmony import */ var _line_saw_special_spans_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../line/saw_special_spans.js */ "./node_modules/codemirror/src/line/saw_special_spans.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");






// Updates the display.view data structure for a given change to the
// document. From and to are in pre-change coordinates. Lendiff is
// the amount of lines added or subtracted by the change. This is
// used for changes that span multiple lines, or change the way
// lines are divided into visual lines. regLineChange (below)
// registers single-line changes.
function regChange(cm, from, to, lendiff) {
  if (from == null) from = cm.doc.first
  if (to == null) to = cm.doc.first + cm.doc.size
  if (!lendiff) lendiff = 0

  let display = cm.display
  if (lendiff && to < display.viewTo &&
      (display.updateLineNumbers == null || display.updateLineNumbers > from))
    display.updateLineNumbers = from

  cm.curOp.viewChanged = true

  if (from >= display.viewTo) { // Change after
    if (_line_saw_special_spans_js__WEBPACK_IMPORTED_MODULE_1__.sawCollapsedSpans && (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_2__.visualLineNo)(cm.doc, from) < display.viewTo)
      resetView(cm)
  } else if (to <= display.viewFrom) { // Change before
    if (_line_saw_special_spans_js__WEBPACK_IMPORTED_MODULE_1__.sawCollapsedSpans && (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_2__.visualLineEndNo)(cm.doc, to + lendiff) > display.viewFrom) {
      resetView(cm)
    } else {
      display.viewFrom += lendiff
      display.viewTo += lendiff
    }
  } else if (from <= display.viewFrom && to >= display.viewTo) { // Full overlap
    resetView(cm)
  } else if (from <= display.viewFrom) { // Top overlap
    let cut = viewCuttingPoint(cm, to, to + lendiff, 1)
    if (cut) {
      display.view = display.view.slice(cut.index)
      display.viewFrom = cut.lineN
      display.viewTo += lendiff
    } else {
      resetView(cm)
    }
  } else if (to >= display.viewTo) { // Bottom overlap
    let cut = viewCuttingPoint(cm, from, from, -1)
    if (cut) {
      display.view = display.view.slice(0, cut.index)
      display.viewTo = cut.lineN
    } else {
      resetView(cm)
    }
  } else { // Gap in the middle
    let cutTop = viewCuttingPoint(cm, from, from, -1)
    let cutBot = viewCuttingPoint(cm, to, to + lendiff, 1)
    if (cutTop && cutBot) {
      display.view = display.view.slice(0, cutTop.index)
        .concat((0,_line_line_data_js__WEBPACK_IMPORTED_MODULE_0__.buildViewArray)(cm, cutTop.lineN, cutBot.lineN))
        .concat(display.view.slice(cutBot.index))
      display.viewTo += lendiff
    } else {
      resetView(cm)
    }
  }

  let ext = display.externalMeasured
  if (ext) {
    if (to < ext.lineN)
      ext.lineN += lendiff
    else if (from < ext.lineN + ext.size)
      display.externalMeasured = null
  }
}

// Register a change to a single line. Type must be one of "text",
// "gutter", "class", "widget"
function regLineChange(cm, line, type) {
  cm.curOp.viewChanged = true
  let display = cm.display, ext = cm.display.externalMeasured
  if (ext && line >= ext.lineN && line < ext.lineN + ext.size)
    display.externalMeasured = null

  if (line < display.viewFrom || line >= display.viewTo) return
  let lineView = display.view[(0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.findViewIndex)(cm, line)]
  if (lineView.node == null) return
  let arr = lineView.changes || (lineView.changes = [])
  if ((0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.indexOf)(arr, type) == -1) arr.push(type)
}

// Clear the view.
function resetView(cm) {
  cm.display.viewFrom = cm.display.viewTo = cm.doc.first
  cm.display.view = []
  cm.display.viewOffset = 0
}

function viewCuttingPoint(cm, oldN, newN, dir) {
  let index = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.findViewIndex)(cm, oldN), diff, view = cm.display.view
  if (!_line_saw_special_spans_js__WEBPACK_IMPORTED_MODULE_1__.sawCollapsedSpans || newN == cm.doc.first + cm.doc.size)
    return {index: index, lineN: newN}
  let n = cm.display.viewFrom
  for (let i = 0; i < index; i++)
    n += view[i].size
  if (n != oldN) {
    if (dir > 0) {
      if (index == view.length - 1) return null
      diff = (n + view[index].size) - oldN
      index++
    } else {
      diff = n - oldN
    }
    oldN += diff; newN += diff
  }
  while ((0,_line_spans_js__WEBPACK_IMPORTED_MODULE_2__.visualLineNo)(cm.doc, newN) != newN) {
    if (index == (dir < 0 ? 0 : view.length - 1)) return null
    newN += dir * view[index - (dir < 0 ? 1 : 0)].size
    index += dir
  }
  return {index: index, lineN: newN}
}

// Force the view to cover a given range, adding empty view element
// or clipping off existing ones as needed.
function adjustView(cm, from, to) {
  let display = cm.display, view = display.view
  if (view.length == 0 || from >= display.viewTo || to <= display.viewFrom) {
    display.view = (0,_line_line_data_js__WEBPACK_IMPORTED_MODULE_0__.buildViewArray)(cm, from, to)
    display.viewFrom = from
  } else {
    if (display.viewFrom > from)
      display.view = (0,_line_line_data_js__WEBPACK_IMPORTED_MODULE_0__.buildViewArray)(cm, from, display.viewFrom).concat(display.view)
    else if (display.viewFrom < from)
      display.view = display.view.slice((0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.findViewIndex)(cm, from))
    display.viewFrom = from
    if (display.viewTo < to)
      display.view = display.view.concat((0,_line_line_data_js__WEBPACK_IMPORTED_MODULE_0__.buildViewArray)(cm, display.viewTo, to))
    else if (display.viewTo > to)
      display.view = display.view.slice(0, (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_3__.findViewIndex)(cm, to))
  }
  display.viewTo = to
}

// Count the number of lines in the view whose DOM representation is
// out of date (or nonexistent).
function countDirtyView(cm) {
  let view = cm.display.view, dirty = 0
  for (let i = 0; i < view.length; i++) {
    let lineView = view[i]
    if (!lineView.hidden && (!lineView.node || lineView.changes)) ++dirty
  }
  return dirty
}


/***/ }),

/***/ "./node_modules/codemirror/src/edit/commands.js":
/*!******************************************************!*\
  !*** ./node_modules/codemirror/src/edit/commands.js ***!
  \******************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "commands": () => (/* binding */ commands)
/* harmony export */ });
/* harmony import */ var _deleteNearSelection_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./deleteNearSelection.js */ "./node_modules/codemirror/src/edit/deleteNearSelection.js");
/* harmony import */ var _display_operations_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../display/operations.js */ "./node_modules/codemirror/src/display/operations.js");
/* harmony import */ var _display_scrolling_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../display/scrolling.js */ "./node_modules/codemirror/src/display/scrolling.js");
/* harmony import */ var _input_movement_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../input/movement.js */ "./node_modules/codemirror/src/input/movement.js");
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _model_selection_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../model/selection.js */ "./node_modules/codemirror/src/model/selection.js");
/* harmony import */ var _model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ../model/selection_updates.js */ "./node_modules/codemirror/src/model/selection_updates.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _util_bidi_js__WEBPACK_IMPORTED_MODULE_10__ = __webpack_require__(/*! ../util/bidi.js */ "./node_modules/codemirror/src/util/bidi.js");












// Commands are parameter-less actions that can be performed on an
// editor, mostly used for keybindings.
let commands = {
  selectAll: _model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.selectAll,
  singleSelection: cm => cm.setSelection(cm.getCursor("anchor"), cm.getCursor("head"), _util_misc_js__WEBPACK_IMPORTED_MODULE_9__.sel_dontScroll),
  killLine: cm => (0,_deleteNearSelection_js__WEBPACK_IMPORTED_MODULE_0__.deleteNearSelection)(cm, range => {
    if (range.empty()) {
      let len = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_6__.getLine)(cm.doc, range.head.line).text.length
      if (range.head.ch == len && range.head.line < cm.lastLine())
        return {from: range.head, to: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(range.head.line + 1, 0)}
      else
        return {from: range.head, to: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(range.head.line, len)}
    } else {
      return {from: range.from(), to: range.to()}
    }
  }),
  deleteLine: cm => (0,_deleteNearSelection_js__WEBPACK_IMPORTED_MODULE_0__.deleteNearSelection)(cm, range => ({
    from: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(range.from().line, 0),
    to: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.clipPos)(cm.doc, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(range.to().line + 1, 0))
  })),
  delLineLeft: cm => (0,_deleteNearSelection_js__WEBPACK_IMPORTED_MODULE_0__.deleteNearSelection)(cm, range => ({
    from: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(range.from().line, 0), to: range.from()
  })),
  delWrappedLineLeft: cm => (0,_deleteNearSelection_js__WEBPACK_IMPORTED_MODULE_0__.deleteNearSelection)(cm, range => {
    let top = cm.charCoords(range.head, "div").top + 5
    let leftPos = cm.coordsChar({left: 0, top: top}, "div")
    return {from: leftPos, to: range.from()}
  }),
  delWrappedLineRight: cm => (0,_deleteNearSelection_js__WEBPACK_IMPORTED_MODULE_0__.deleteNearSelection)(cm, range => {
    let top = cm.charCoords(range.head, "div").top + 5
    let rightPos = cm.coordsChar({left: cm.display.lineDiv.offsetWidth + 100, top: top}, "div")
    return {from: range.from(), to: rightPos }
  }),
  undo: cm => cm.undo(),
  redo: cm => cm.redo(),
  undoSelection: cm => cm.undoSelection(),
  redoSelection: cm => cm.redoSelection(),
  goDocStart: cm => cm.extendSelection((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(cm.firstLine(), 0)),
  goDocEnd: cm => cm.extendSelection((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(cm.lastLine())),
  goLineStart: cm => cm.extendSelectionsBy(range => lineStart(cm, range.head.line),
    {origin: "+move", bias: 1}
  ),
  goLineStartSmart: cm => cm.extendSelectionsBy(range => lineStartSmart(cm, range.head),
    {origin: "+move", bias: 1}
  ),
  goLineEnd: cm => cm.extendSelectionsBy(range => lineEnd(cm, range.head.line),
    {origin: "+move", bias: -1}
  ),
  goLineRight: cm => cm.extendSelectionsBy(range => {
    let top = cm.cursorCoords(range.head, "div").top + 5
    return cm.coordsChar({left: cm.display.lineDiv.offsetWidth + 100, top: top}, "div")
  }, _util_misc_js__WEBPACK_IMPORTED_MODULE_9__.sel_move),
  goLineLeft: cm => cm.extendSelectionsBy(range => {
    let top = cm.cursorCoords(range.head, "div").top + 5
    return cm.coordsChar({left: 0, top: top}, "div")
  }, _util_misc_js__WEBPACK_IMPORTED_MODULE_9__.sel_move),
  goLineLeftSmart: cm => cm.extendSelectionsBy(range => {
    let top = cm.cursorCoords(range.head, "div").top + 5
    let pos = cm.coordsChar({left: 0, top: top}, "div")
    if (pos.ch < cm.getLine(pos.line).search(/\S/)) return lineStartSmart(cm, range.head)
    return pos
  }, _util_misc_js__WEBPACK_IMPORTED_MODULE_9__.sel_move),
  goLineUp: cm => cm.moveV(-1, "line"),
  goLineDown: cm => cm.moveV(1, "line"),
  goPageUp: cm => cm.moveV(-1, "page"),
  goPageDown: cm => cm.moveV(1, "page"),
  goCharLeft: cm => cm.moveH(-1, "char"),
  goCharRight: cm => cm.moveH(1, "char"),
  goColumnLeft: cm => cm.moveH(-1, "column"),
  goColumnRight: cm => cm.moveH(1, "column"),
  goWordLeft: cm => cm.moveH(-1, "word"),
  goGroupRight: cm => cm.moveH(1, "group"),
  goGroupLeft: cm => cm.moveH(-1, "group"),
  goWordRight: cm => cm.moveH(1, "word"),
  delCharBefore: cm => cm.deleteH(-1, "codepoint"),
  delCharAfter: cm => cm.deleteH(1, "char"),
  delWordBefore: cm => cm.deleteH(-1, "word"),
  delWordAfter: cm => cm.deleteH(1, "word"),
  delGroupBefore: cm => cm.deleteH(-1, "group"),
  delGroupAfter: cm => cm.deleteH(1, "group"),
  indentAuto: cm => cm.indentSelection("smart"),
  indentMore: cm => cm.indentSelection("add"),
  indentLess: cm => cm.indentSelection("subtract"),
  insertTab: cm => cm.replaceSelection("\t"),
  insertSoftTab: cm => {
    let spaces = [], ranges = cm.listSelections(), tabSize = cm.options.tabSize
    for (let i = 0; i < ranges.length; i++) {
      let pos = ranges[i].from()
      let col = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.countColumn)(cm.getLine(pos.line), pos.ch, tabSize)
      spaces.push((0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.spaceStr)(tabSize - col % tabSize))
    }
    cm.replaceSelections(spaces)
  },
  defaultTab: cm => {
    if (cm.somethingSelected()) cm.indentSelection("add")
    else cm.execCommand("insertTab")
  },
  // Swap the two chars left and right of each selection's head.
  // Move cursor behind the two swapped characters afterwards.
  //
  // Doesn't consider line feeds a character.
  // Doesn't scan more than one line above to find a character.
  // Doesn't do anything on an empty line.
  // Doesn't do anything with non-empty selections.
  transposeChars: cm => (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_1__.runInOp)(cm, () => {
    let ranges = cm.listSelections(), newSel = []
    for (let i = 0; i < ranges.length; i++) {
      if (!ranges[i].empty()) continue
      let cur = ranges[i].head, line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_6__.getLine)(cm.doc, cur.line).text
      if (line) {
        if (cur.ch == line.length) cur = new _line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos(cur.line, cur.ch - 1)
        if (cur.ch > 0) {
          cur = new _line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos(cur.line, cur.ch + 1)
          cm.replaceRange(line.charAt(cur.ch - 1) + line.charAt(cur.ch - 2),
                          (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(cur.line, cur.ch - 2), cur, "+transpose")
        } else if (cur.line > cm.doc.first) {
          let prev = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_6__.getLine)(cm.doc, cur.line - 1).text
          if (prev) {
            cur = new _line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos(cur.line, 1)
            cm.replaceRange(line.charAt(0) + cm.doc.lineSeparator() +
                            prev.charAt(prev.length - 1),
                            (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(cur.line - 1, prev.length - 1), cur, "+transpose")
          }
        }
      }
      newSel.push(new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(cur, cur))
    }
    cm.setSelections(newSel)
  }),
  newlineAndIndent: cm => (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_1__.runInOp)(cm, () => {
    let sels = cm.listSelections()
    for (let i = sels.length - 1; i >= 0; i--)
      cm.replaceRange(cm.doc.lineSeparator(), sels[i].anchor, sels[i].head, "+input")
    sels = cm.listSelections()
    for (let i = 0; i < sels.length; i++)
      cm.indentLine(sels[i].from().line, null, true)
    ;(0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_2__.ensureCursorVisible)(cm)
  }),
  openLine: cm => cm.replaceSelection("\n", "start"),
  toggleOverwrite: cm => cm.toggleOverwrite()
}


function lineStart(cm, lineN) {
  let line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_6__.getLine)(cm.doc, lineN)
  let visual = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_5__.visualLine)(line)
  if (visual != line) lineN = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_6__.lineNo)(visual)
  return (0,_input_movement_js__WEBPACK_IMPORTED_MODULE_3__.endOfLine)(true, cm, visual, lineN, 1)
}
function lineEnd(cm, lineN) {
  let line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_6__.getLine)(cm.doc, lineN)
  let visual = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_5__.visualLineEnd)(line)
  if (visual != line) lineN = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_6__.lineNo)(visual)
  return (0,_input_movement_js__WEBPACK_IMPORTED_MODULE_3__.endOfLine)(true, cm, line, lineN, -1)
}
function lineStartSmart(cm, pos) {
  let start = lineStart(cm, pos.line)
  let line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_6__.getLine)(cm.doc, start.line)
  let order = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_10__.getOrder)(line, cm.doc.direction)
  if (!order || order[0].level == 0) {
    let firstNonWS = Math.max(start.ch, line.text.search(/\S/))
    let inWS = pos.line == start.line && pos.ch <= firstNonWS && pos.ch
    return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(start.line, inWS ? 0 : firstNonWS, start.sticky)
  }
  return start
}


/***/ }),

/***/ "./node_modules/codemirror/src/edit/deleteNearSelection.js":
/*!*****************************************************************!*\
  !*** ./node_modules/codemirror/src/edit/deleteNearSelection.js ***!
  \*****************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "deleteNearSelection": () => (/* binding */ deleteNearSelection)
/* harmony export */ });
/* harmony import */ var _display_operations_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../display/operations.js */ "./node_modules/codemirror/src/display/operations.js");
/* harmony import */ var _display_scrolling_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../display/scrolling.js */ "./node_modules/codemirror/src/display/scrolling.js");
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _model_changes_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../model/changes.js */ "./node_modules/codemirror/src/model/changes.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");






// Helper for deleting text near the selection(s), used to implement
// backspace, delete, and similar functionality.
function deleteNearSelection(cm, compute) {
  let ranges = cm.doc.sel.ranges, kill = []
  // Build up a set of ranges to kill first, merging overlapping
  // ranges.
  for (let i = 0; i < ranges.length; i++) {
    let toKill = compute(ranges[i])
    while (kill.length && (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.cmp)(toKill.from, (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(kill).to) <= 0) {
      let replaced = kill.pop()
      if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.cmp)(replaced.from, toKill.from) < 0) {
        toKill.from = replaced.from
        break
      }
    }
    kill.push(toKill)
  }
  // Next, remove those actual ranges.
  (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_0__.runInOp)(cm, () => {
    for (let i = kill.length - 1; i >= 0; i--)
      (0,_model_changes_js__WEBPACK_IMPORTED_MODULE_3__.replaceRange)(cm.doc, "", kill[i].from, kill[i].to, "+delete")
    ;(0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_1__.ensureCursorVisible)(cm)
  })
}


/***/ }),

/***/ "./node_modules/codemirror/src/edit/key_events.js":
/*!********************************************************!*\
  !*** ./node_modules/codemirror/src/edit/key_events.js ***!
  \********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "dispatchKey": () => (/* binding */ dispatchKey),
/* harmony export */   "onKeyDown": () => (/* binding */ onKeyDown),
/* harmony export */   "onKeyUp": () => (/* binding */ onKeyUp),
/* harmony export */   "onKeyPress": () => (/* binding */ onKeyPress)
/* harmony export */ });
/* harmony import */ var _util_operation_group_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../util/operation_group.js */ "./node_modules/codemirror/src/util/operation_group.js");
/* harmony import */ var _display_selection_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../display/selection.js */ "./node_modules/codemirror/src/display/selection.js");
/* harmony import */ var _input_keymap_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../input/keymap.js */ "./node_modules/codemirror/src/input/keymap.js");
/* harmony import */ var _measurement_widgets_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../measurement/widgets.js */ "./node_modules/codemirror/src/measurement/widgets.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _util_feature_detection_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../util/feature_detection.js */ "./node_modules/codemirror/src/util/feature_detection.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _commands_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ./commands.js */ "./node_modules/codemirror/src/edit/commands.js");












// Run a handler that was bound to a key.
function doHandleBinding(cm, bound, dropShift) {
  if (typeof bound == "string") {
    bound = _commands_js__WEBPACK_IMPORTED_MODULE_9__.commands[bound]
    if (!bound) return false
  }
  // Ensure previous input has been read, so that the handler sees a
  // consistent view of the document
  cm.display.input.ensurePolled()
  let prevShift = cm.display.shift, done = false
  try {
    if (cm.isReadOnly()) cm.state.suppressEdits = true
    if (dropShift) cm.display.shift = false
    done = bound(cm) != _util_misc_js__WEBPACK_IMPORTED_MODULE_8__.Pass
  } finally {
    cm.display.shift = prevShift
    cm.state.suppressEdits = false
  }
  return done
}

function lookupKeyForEditor(cm, name, handle) {
  for (let i = 0; i < cm.state.keyMaps.length; i++) {
    let result = (0,_input_keymap_js__WEBPACK_IMPORTED_MODULE_2__.lookupKey)(name, cm.state.keyMaps[i], handle, cm)
    if (result) return result
  }
  return (cm.options.extraKeys && (0,_input_keymap_js__WEBPACK_IMPORTED_MODULE_2__.lookupKey)(name, cm.options.extraKeys, handle, cm))
    || (0,_input_keymap_js__WEBPACK_IMPORTED_MODULE_2__.lookupKey)(name, cm.options.keyMap, handle, cm)
}

// Note that, despite the name, this function is also used to check
// for bound mouse clicks.

let stopSeq = new _util_misc_js__WEBPACK_IMPORTED_MODULE_8__.Delayed

function dispatchKey(cm, name, e, handle) {
  let seq = cm.state.keySeq
  if (seq) {
    if ((0,_input_keymap_js__WEBPACK_IMPORTED_MODULE_2__.isModifierKey)(name)) return "handled"
    if (/\'$/.test(name))
      cm.state.keySeq = null
    else
      stopSeq.set(50, () => {
        if (cm.state.keySeq == seq) {
          cm.state.keySeq = null
          cm.display.input.reset()
        }
      })
    if (dispatchKeyInner(cm, seq + " " + name, e, handle)) return true
  }
  return dispatchKeyInner(cm, name, e, handle)
}

function dispatchKeyInner(cm, name, e, handle) {
  let result = lookupKeyForEditor(cm, name, handle)

  if (result == "multi")
    cm.state.keySeq = name
  if (result == "handled")
    (0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_0__.signalLater)(cm, "keyHandled", cm, name, e)

  if (result == "handled" || result == "multi") {
    (0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.e_preventDefault)(e)
    ;(0,_display_selection_js__WEBPACK_IMPORTED_MODULE_1__.restartBlink)(cm)
  }

  return !!result
}

// Handle a key from the keydown event.
function handleKeyBinding(cm, e) {
  let name = (0,_input_keymap_js__WEBPACK_IMPORTED_MODULE_2__.keyName)(e, true)
  if (!name) return false

  if (e.shiftKey && !cm.state.keySeq) {
    // First try to resolve full name (including 'Shift-'). Failing
    // that, see if there is a cursor-motion command (starting with
    // 'go') bound to the keyname without 'Shift-'.
    return dispatchKey(cm, "Shift-" + name, e, b => doHandleBinding(cm, b, true))
        || dispatchKey(cm, name, e, b => {
             if (typeof b == "string" ? /^go[A-Z]/.test(b) : b.motion)
               return doHandleBinding(cm, b)
           })
  } else {
    return dispatchKey(cm, name, e, b => doHandleBinding(cm, b))
  }
}

// Handle a key from the keypress event
function handleCharBinding(cm, e, ch) {
  return dispatchKey(cm, "'" + ch + "'", e, b => doHandleBinding(cm, b, true))
}

let lastStoppedKey = null
function onKeyDown(e) {
  let cm = this
  if (e.target && e.target != cm.display.input.getField()) return
  cm.curOp.focus = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.activeElt)()
  if ((0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.signalDOMEvent)(cm, e)) return
  // IE does strange things with escape.
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_4__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_4__.ie_version < 11 && e.keyCode == 27) e.returnValue = false
  let code = e.keyCode
  cm.display.shift = code == 16 || e.shiftKey
  let handled = handleKeyBinding(cm, e)
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_4__.presto) {
    lastStoppedKey = handled ? code : null
    // Opera has no cut event... we try to at least catch the key combo
    if (!handled && code == 88 && !_util_feature_detection_js__WEBPACK_IMPORTED_MODULE_7__.hasCopyEvent && (_util_browser_js__WEBPACK_IMPORTED_MODULE_4__.mac ? e.metaKey : e.ctrlKey))
      cm.replaceSelection("", null, "cut")
  }
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_4__.gecko && !_util_browser_js__WEBPACK_IMPORTED_MODULE_4__.mac && !handled && code == 46 && e.shiftKey && !e.ctrlKey && document.execCommand)
    document.execCommand("cut")

  // Turn mouse into crosshair when Alt is held on Mac.
  if (code == 18 && !/\bCodeMirror-crosshair\b/.test(cm.display.lineDiv.className))
    showCrossHair(cm)
}

function showCrossHair(cm) {
  let lineDiv = cm.display.lineDiv
  ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.addClass)(lineDiv, "CodeMirror-crosshair")

  function up(e) {
    if (e.keyCode == 18 || !e.altKey) {
      (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_5__.rmClass)(lineDiv, "CodeMirror-crosshair")
      ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.off)(document, "keyup", up)
      ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.off)(document, "mouseover", up)
    }
  }
  (0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.on)(document, "keyup", up)
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.on)(document, "mouseover", up)
}

function onKeyUp(e) {
  if (e.keyCode == 16) this.doc.sel.shift = false
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.signalDOMEvent)(this, e)
}

function onKeyPress(e) {
  let cm = this
  if (e.target && e.target != cm.display.input.getField()) return
  if ((0,_measurement_widgets_js__WEBPACK_IMPORTED_MODULE_3__.eventInWidget)(cm.display, e) || (0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.signalDOMEvent)(cm, e) || e.ctrlKey && !e.altKey || _util_browser_js__WEBPACK_IMPORTED_MODULE_4__.mac && e.metaKey) return
  let keyCode = e.keyCode, charCode = e.charCode
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_4__.presto && keyCode == lastStoppedKey) {lastStoppedKey = null; (0,_util_event_js__WEBPACK_IMPORTED_MODULE_6__.e_preventDefault)(e); return}
  if ((_util_browser_js__WEBPACK_IMPORTED_MODULE_4__.presto && (!e.which || e.which < 10)) && handleKeyBinding(cm, e)) return
  let ch = String.fromCharCode(charCode == null ? keyCode : charCode)
  // Some browsers fire keypress events for backspace
  if (ch == "\x08") return
  if (handleCharBinding(cm, e, ch)) return
  cm.display.input.onKeyPress(e)
}


/***/ }),

/***/ "./node_modules/codemirror/src/edit/methods.js":
/*!*****************************************************!*\
  !*** ./node_modules/codemirror/src/edit/methods.js ***!
  \*****************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "default": () => (/* export default binding */ __WEBPACK_DEFAULT_EXPORT__)
/* harmony export */ });
/* harmony import */ var _deleteNearSelection_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./deleteNearSelection.js */ "./node_modules/codemirror/src/edit/deleteNearSelection.js");
/* harmony import */ var _commands_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ./commands.js */ "./node_modules/codemirror/src/edit/commands.js");
/* harmony import */ var _model_document_data_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../model/document_data.js */ "./node_modules/codemirror/src/model/document_data.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _line_highlight_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../line/highlight.js */ "./node_modules/codemirror/src/line/highlight.js");
/* harmony import */ var _input_indent_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../input/indent.js */ "./node_modules/codemirror/src/input/indent.js");
/* harmony import */ var _input_input_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../input/input.js */ "./node_modules/codemirror/src/input/input.js");
/* harmony import */ var _key_events_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ./key_events.js */ "./node_modules/codemirror/src/edit/key_events.js");
/* harmony import */ var _mouse_events_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ./mouse_events.js */ "./node_modules/codemirror/src/edit/mouse_events.js");
/* harmony import */ var _input_keymap_js__WEBPACK_IMPORTED_MODULE_10__ = __webpack_require__(/*! ../input/keymap.js */ "./node_modules/codemirror/src/input/keymap.js");
/* harmony import */ var _input_movement_js__WEBPACK_IMPORTED_MODULE_11__ = __webpack_require__(/*! ../input/movement.js */ "./node_modules/codemirror/src/input/movement.js");
/* harmony import */ var _display_operations_js__WEBPACK_IMPORTED_MODULE_12__ = __webpack_require__(/*! ../display/operations.js */ "./node_modules/codemirror/src/display/operations.js");
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_13__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _model_selection_js__WEBPACK_IMPORTED_MODULE_15__ = __webpack_require__(/*! ../model/selection.js */ "./node_modules/codemirror/src/model/selection.js");
/* harmony import */ var _model_selection_updates_js__WEBPACK_IMPORTED_MODULE_16__ = __webpack_require__(/*! ../model/selection_updates.js */ "./node_modules/codemirror/src/model/selection_updates.js");
/* harmony import */ var _display_scrolling_js__WEBPACK_IMPORTED_MODULE_17__ = __webpack_require__(/*! ../display/scrolling.js */ "./node_modules/codemirror/src/display/scrolling.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_18__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _display_update_display_js__WEBPACK_IMPORTED_MODULE_19__ = __webpack_require__(/*! ../display/update_display.js */ "./node_modules/codemirror/src/display/update_display.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_20__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _util_operation_group_js__WEBPACK_IMPORTED_MODULE_21__ = __webpack_require__(/*! ../util/operation_group.js */ "./node_modules/codemirror/src/util/operation_group.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_22__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _display_view_tracking_js__WEBPACK_IMPORTED_MODULE_23__ = __webpack_require__(/*! ../display/view_tracking.js */ "./node_modules/codemirror/src/display/view_tracking.js");

























// The publicly visible API. Note that methodOp(f) means
// 'wrap f in an operation, performed on its `this` parameter'.

// This is not the complete set of editor methods. Most of the
// methods defined on the Doc type are also injected into
// CodeMirror.prototype, for backwards compatibility and
// convenience.

/* harmony default export */ function __WEBPACK_DEFAULT_EXPORT__(CodeMirror) {
  let optionHandlers = CodeMirror.optionHandlers

  let helpers = CodeMirror.helpers = {}

  CodeMirror.prototype = {
    constructor: CodeMirror,
    focus: function(){window.focus(); this.display.input.focus()},

    setOption: function(option, value) {
      let options = this.options, old = options[option]
      if (options[option] == value && option != "mode") return
      options[option] = value
      if (optionHandlers.hasOwnProperty(option))
        (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.operation)(this, optionHandlers[option])(this, value, old)
      ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.signal)(this, "optionChange", this, option)
    },

    getOption: function(option) {return this.options[option]},
    getDoc: function() {return this.doc},

    addKeyMap: function(map, bottom) {
      this.state.keyMaps[bottom ? "push" : "unshift"]((0,_input_keymap_js__WEBPACK_IMPORTED_MODULE_10__.getKeyMap)(map))
    },
    removeKeyMap: function(map) {
      let maps = this.state.keyMaps
      for (let i = 0; i < maps.length; ++i)
        if (maps[i] == map || maps[i].name == map) {
          maps.splice(i, 1)
          return true
        }
    },

    addOverlay: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(spec, options) {
      let mode = spec.token ? spec : CodeMirror.getMode(this.options, spec)
      if (mode.startState) throw new Error("Overlays may not be stateful.")
      ;(0,_util_misc_js__WEBPACK_IMPORTED_MODULE_20__.insertSorted)(this.state.overlays,
                   {mode: mode, modeSpec: spec, opaque: options && options.opaque,
                    priority: (options && options.priority) || 0},
                   overlay => overlay.priority)
      this.state.modeGen++
      ;(0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_23__.regChange)(this)
    }),
    removeOverlay: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(spec) {
      let overlays = this.state.overlays
      for (let i = 0; i < overlays.length; ++i) {
        let cur = overlays[i].modeSpec
        if (cur == spec || typeof spec == "string" && cur.name == spec) {
          overlays.splice(i, 1)
          this.state.modeGen++
          ;(0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_23__.regChange)(this)
          return
        }
      }
    }),

    indentLine: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(n, dir, aggressive) {
      if (typeof dir != "string" && typeof dir != "number") {
        if (dir == null) dir = this.options.smartIndent ? "smart" : "prev"
        else dir = dir ? "add" : "subtract"
      }
      if ((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_22__.isLine)(this.doc, n)) (0,_input_indent_js__WEBPACK_IMPORTED_MODULE_6__.indentLine)(this, n, dir, aggressive)
    }),
    indentSelection: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(how) {
      let ranges = this.doc.sel.ranges, end = -1
      for (let i = 0; i < ranges.length; i++) {
        let range = ranges[i]
        if (!range.empty()) {
          let from = range.from(), to = range.to()
          let start = Math.max(end, from.line)
          end = Math.min(this.lastLine(), to.line - (to.ch ? 0 : 1)) + 1
          for (let j = start; j < end; ++j)
            (0,_input_indent_js__WEBPACK_IMPORTED_MODULE_6__.indentLine)(this, j, how)
          let newRanges = this.doc.sel.ranges
          if (from.ch == 0 && ranges.length == newRanges.length && newRanges[i].from().ch > 0)
            (0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_16__.replaceOneSelection)(this.doc, i, new _model_selection_js__WEBPACK_IMPORTED_MODULE_15__.Range(from, newRanges[i].to()), _util_misc_js__WEBPACK_IMPORTED_MODULE_20__.sel_dontScroll)
        } else if (range.head.line > end) {
          (0,_input_indent_js__WEBPACK_IMPORTED_MODULE_6__.indentLine)(this, range.head.line, how, true)
          end = range.head.line
          if (i == this.doc.sel.primIndex) (0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_17__.ensureCursorVisible)(this)
        }
      }
    }),

    // Fetch the parser token for a given character. Useful for hacks
    // that want to inspect the mode state (say, for completion).
    getTokenAt: function(pos, precise) {
      return (0,_line_highlight_js__WEBPACK_IMPORTED_MODULE_5__.takeToken)(this, pos, precise)
    },

    getLineTokens: function(line, precise) {
      return (0,_line_highlight_js__WEBPACK_IMPORTED_MODULE_5__.takeToken)(this, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.Pos)(line), precise, true)
    },

    getTokenTypeAt: function(pos) {
      pos = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.clipPos)(this.doc, pos)
      let styles = (0,_line_highlight_js__WEBPACK_IMPORTED_MODULE_5__.getLineStyles)(this, (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_22__.getLine)(this.doc, pos.line))
      let before = 0, after = (styles.length - 1) / 2, ch = pos.ch
      let type
      if (ch == 0) type = styles[2]
      else for (;;) {
        let mid = (before + after) >> 1
        if ((mid ? styles[mid * 2 - 1] : 0) >= ch) after = mid
        else if (styles[mid * 2 + 1] < ch) before = mid + 1
        else { type = styles[mid * 2 + 2]; break }
      }
      let cut = type ? type.indexOf("overlay ") : -1
      return cut < 0 ? type : cut == 0 ? null : type.slice(0, cut - 1)
    },

    getModeAt: function(pos) {
      let mode = this.doc.mode
      if (!mode.innerMode) return mode
      return CodeMirror.innerMode(mode, this.getTokenAt(pos).state).mode
    },

    getHelper: function(pos, type) {
      return this.getHelpers(pos, type)[0]
    },

    getHelpers: function(pos, type) {
      let found = []
      if (!helpers.hasOwnProperty(type)) return found
      let help = helpers[type], mode = this.getModeAt(pos)
      if (typeof mode[type] == "string") {
        if (help[mode[type]]) found.push(help[mode[type]])
      } else if (mode[type]) {
        for (let i = 0; i < mode[type].length; i++) {
          let val = help[mode[type][i]]
          if (val) found.push(val)
        }
      } else if (mode.helperType && help[mode.helperType]) {
        found.push(help[mode.helperType])
      } else if (help[mode.name]) {
        found.push(help[mode.name])
      }
      for (let i = 0; i < help._global.length; i++) {
        let cur = help._global[i]
        if (cur.pred(mode, this) && (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_20__.indexOf)(found, cur.val) == -1)
          found.push(cur.val)
      }
      return found
    },

    getStateAfter: function(line, precise) {
      let doc = this.doc
      line = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.clipLine)(doc, line == null ? doc.first + doc.size - 1: line)
      return (0,_line_highlight_js__WEBPACK_IMPORTED_MODULE_5__.getContextBefore)(this, line + 1, precise).state
    },

    cursorCoords: function(start, mode) {
      let pos, range = this.doc.sel.primary()
      if (start == null) pos = range.head
      else if (typeof start == "object") pos = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.clipPos)(this.doc, start)
      else pos = start ? range.from() : range.to()
      return (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.cursorCoords)(this, pos, mode || "page")
    },

    charCoords: function(pos, mode) {
      return (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.charCoords)(this, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.clipPos)(this.doc, pos), mode || "page")
    },

    coordsChar: function(coords, mode) {
      coords = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.fromCoordSystem)(this, coords, mode || "page")
      return (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.coordsChar)(this, coords.left, coords.top)
    },

    lineAtHeight: function(height, mode) {
      height = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.fromCoordSystem)(this, {top: height, left: 0}, mode || "page").top
      return (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_22__.lineAtHeight)(this.doc, height + this.display.viewOffset)
    },
    heightAtLine: function(line, mode, includeWidgets) {
      let end = false, lineObj
      if (typeof line == "number") {
        let last = this.doc.first + this.doc.size - 1
        if (line < this.doc.first) line = this.doc.first
        else if (line > last) { line = last; end = true }
        lineObj = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_22__.getLine)(this.doc, line)
      } else {
        lineObj = line
      }
      return (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.intoCoordSystem)(this, lineObj, {top: 0, left: 0}, mode || "page", includeWidgets || end).top +
        (end ? this.doc.height - (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_18__.heightAtLine)(lineObj) : 0)
    },

    defaultTextHeight: function() { return (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.textHeight)(this.display) },
    defaultCharWidth: function() { return (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.charWidth)(this.display) },

    getViewport: function() { return {from: this.display.viewFrom, to: this.display.viewTo}},

    addWidget: function(pos, node, scroll, vert, horiz) {
      let display = this.display
      pos = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.cursorCoords)(this, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.clipPos)(this.doc, pos))
      let top = pos.bottom, left = pos.left
      node.style.position = "absolute"
      node.setAttribute("cm-ignore-events", "true")
      this.display.input.setUneditable(node)
      display.sizer.appendChild(node)
      if (vert == "over") {
        top = pos.top
      } else if (vert == "above" || vert == "near") {
        let vspace = Math.max(display.wrapper.clientHeight, this.doc.height),
        hspace = Math.max(display.sizer.clientWidth, display.lineSpace.clientWidth)
        // Default to positioning above (if specified and possible); otherwise default to positioning below
        if ((vert == 'above' || pos.bottom + node.offsetHeight > vspace) && pos.top > node.offsetHeight)
          top = pos.top - node.offsetHeight
        else if (pos.bottom + node.offsetHeight <= vspace)
          top = pos.bottom
        if (left + node.offsetWidth > hspace)
          left = hspace - node.offsetWidth
      }
      node.style.top = top + "px"
      node.style.left = node.style.right = ""
      if (horiz == "right") {
        left = display.sizer.clientWidth - node.offsetWidth
        node.style.right = "0px"
      } else {
        if (horiz == "left") left = 0
        else if (horiz == "middle") left = (display.sizer.clientWidth - node.offsetWidth) / 2
        node.style.left = left + "px"
      }
      if (scroll)
        (0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_17__.scrollIntoView)(this, {left, top, right: left + node.offsetWidth, bottom: top + node.offsetHeight})
    },

    triggerOnKeyDown: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(_key_events_js__WEBPACK_IMPORTED_MODULE_8__.onKeyDown),
    triggerOnKeyPress: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(_key_events_js__WEBPACK_IMPORTED_MODULE_8__.onKeyPress),
    triggerOnKeyUp: _key_events_js__WEBPACK_IMPORTED_MODULE_8__.onKeyUp,
    triggerOnMouseDown: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(_mouse_events_js__WEBPACK_IMPORTED_MODULE_9__.onMouseDown),

    execCommand: function(cmd) {
      if (_commands_js__WEBPACK_IMPORTED_MODULE_1__.commands.hasOwnProperty(cmd))
        return _commands_js__WEBPACK_IMPORTED_MODULE_1__.commands[cmd].call(null, this)
    },

    triggerElectric: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(text) { (0,_input_input_js__WEBPACK_IMPORTED_MODULE_7__.triggerElectric)(this, text) }),

    findPosH: function(from, amount, unit, visually) {
      let dir = 1
      if (amount < 0) { dir = -1; amount = -amount }
      let cur = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.clipPos)(this.doc, from)
      for (let i = 0; i < amount; ++i) {
        cur = findPosH(this.doc, cur, dir, unit, visually)
        if (cur.hitSide) break
      }
      return cur
    },

    moveH: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(dir, unit) {
      this.extendSelectionsBy(range => {
        if (this.display.shift || this.doc.extend || range.empty())
          return findPosH(this.doc, range.head, dir, unit, this.options.rtlMoveVisually)
        else
          return dir < 0 ? range.from() : range.to()
      }, _util_misc_js__WEBPACK_IMPORTED_MODULE_20__.sel_move)
    }),

    deleteH: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(dir, unit) {
      let sel = this.doc.sel, doc = this.doc
      if (sel.somethingSelected())
        doc.replaceSelection("", null, "+delete")
      else
        (0,_deleteNearSelection_js__WEBPACK_IMPORTED_MODULE_0__.deleteNearSelection)(this, range => {
          let other = findPosH(doc, range.head, dir, unit, false)
          return dir < 0 ? {from: other, to: range.head} : {from: range.head, to: other}
        })
    }),

    findPosV: function(from, amount, unit, goalColumn) {
      let dir = 1, x = goalColumn
      if (amount < 0) { dir = -1; amount = -amount }
      let cur = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.clipPos)(this.doc, from)
      for (let i = 0; i < amount; ++i) {
        let coords = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.cursorCoords)(this, cur, "div")
        if (x == null) x = coords.left
        else coords.left = x
        cur = findPosV(this, coords, dir, unit)
        if (cur.hitSide) break
      }
      return cur
    },

    moveV: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(dir, unit) {
      let doc = this.doc, goals = []
      let collapse = !this.display.shift && !doc.extend && doc.sel.somethingSelected()
      doc.extendSelectionsBy(range => {
        if (collapse)
          return dir < 0 ? range.from() : range.to()
        let headPos = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.cursorCoords)(this, range.head, "div")
        if (range.goalColumn != null) headPos.left = range.goalColumn
        goals.push(headPos.left)
        let pos = findPosV(this, headPos, dir, unit)
        if (unit == "page" && range == doc.sel.primary())
          (0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_17__.addToScrollTop)(this, (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.charCoords)(this, pos, "div").top - headPos.top)
        return pos
      }, _util_misc_js__WEBPACK_IMPORTED_MODULE_20__.sel_move)
      if (goals.length) for (let i = 0; i < doc.sel.ranges.length; i++)
        doc.sel.ranges[i].goalColumn = goals[i]
    }),

    // Find the word at the given position (as returned by coordsChar).
    findWordAt: function(pos) {
      let doc = this.doc, line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_22__.getLine)(doc, pos.line).text
      let start = pos.ch, end = pos.ch
      if (line) {
        let helper = this.getHelper(pos, "wordChars")
        if ((pos.sticky == "before" || end == line.length) && start) --start; else ++end
        let startChar = line.charAt(start)
        let check = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_20__.isWordChar)(startChar, helper)
          ? ch => (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_20__.isWordChar)(ch, helper)
          : /\s/.test(startChar) ? ch => /\s/.test(ch)
          : ch => (!/\s/.test(ch) && !(0,_util_misc_js__WEBPACK_IMPORTED_MODULE_20__.isWordChar)(ch))
        while (start > 0 && check(line.charAt(start - 1))) --start
        while (end < line.length && check(line.charAt(end))) ++end
      }
      return new _model_selection_js__WEBPACK_IMPORTED_MODULE_15__.Range((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.Pos)(pos.line, start), (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.Pos)(pos.line, end))
    },

    toggleOverwrite: function(value) {
      if (value != null && value == this.state.overwrite) return
      if (this.state.overwrite = !this.state.overwrite)
        (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.addClass)(this.display.cursorDiv, "CodeMirror-overwrite")
      else
        (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.rmClass)(this.display.cursorDiv, "CodeMirror-overwrite")

      ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.signal)(this, "overwriteToggle", this, this.state.overwrite)
    },
    hasFocus: function() { return this.display.input.getField() == (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_3__.activeElt)() },
    isReadOnly: function() { return !!(this.options.readOnly || this.doc.cantEdit) },

    scrollTo: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function (x, y) { (0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_17__.scrollToCoords)(this, x, y) }),
    getScrollInfo: function() {
      let scroller = this.display.scroller
      return {left: scroller.scrollLeft, top: scroller.scrollTop,
              height: scroller.scrollHeight - (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.scrollGap)(this) - this.display.barHeight,
              width: scroller.scrollWidth - (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.scrollGap)(this) - this.display.barWidth,
              clientHeight: (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.displayHeight)(this), clientWidth: (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.displayWidth)(this)}
    },

    scrollIntoView: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(range, margin) {
      if (range == null) {
        range = {from: this.doc.sel.primary().head, to: null}
        if (margin == null) margin = this.options.cursorScrollMargin
      } else if (typeof range == "number") {
        range = {from: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.Pos)(range, 0), to: null}
      } else if (range.from == null) {
        range = {from: range, to: null}
      }
      if (!range.to) range.to = range.from
      range.margin = margin || 0

      if (range.from.line != null) {
        (0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_17__.scrollToRange)(this, range)
      } else {
        (0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_17__.scrollToCoordsRange)(this, range.from, range.to, range.margin)
      }
    }),

    setSize: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(width, height) {
      let interpret = val => typeof val == "number" || /^\d+$/.test(String(val)) ? val + "px" : val
      if (width != null) this.display.wrapper.style.width = interpret(width)
      if (height != null) this.display.wrapper.style.height = interpret(height)
      if (this.options.lineWrapping) (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.clearLineMeasurementCache)(this)
      let lineNo = this.display.viewFrom
      this.doc.iter(lineNo, this.display.viewTo, line => {
        if (line.widgets) for (let i = 0; i < line.widgets.length; i++)
          if (line.widgets[i].noHScroll) { (0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_23__.regLineChange)(this, lineNo, "widget"); break }
        ++lineNo
      })
      this.curOp.forceUpdate = true
      ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.signal)(this, "refresh", this)
    }),

    operation: function(f){return (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.runInOp)(this, f)},
    startOperation: function(){return (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.startOperation)(this)},
    endOperation: function(){return (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.endOperation)(this)},

    refresh: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function() {
      let oldHeight = this.display.cachedTextHeight
      ;(0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_23__.regChange)(this)
      this.curOp.forceUpdate = true
      ;(0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.clearCaches)(this)
      ;(0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_17__.scrollToCoords)(this, this.doc.scrollLeft, this.doc.scrollTop)
      ;(0,_display_update_display_js__WEBPACK_IMPORTED_MODULE_19__.updateGutterSpace)(this.display)
      if (oldHeight == null || Math.abs(oldHeight - (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.textHeight)(this.display)) > .5 || this.options.lineWrapping)
        (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.estimateLineHeights)(this)
      ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.signal)(this, "refresh", this)
    }),

    swapDoc: (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_12__.methodOp)(function(doc) {
      let old = this.doc
      old.cm = null
      // Cancel the current text selection if any (#5821)
      if (this.state.selectingText) this.state.selectingText()
      ;(0,_model_document_data_js__WEBPACK_IMPORTED_MODULE_2__.attachDoc)(this, doc)
      ;(0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.clearCaches)(this)
      this.display.input.reset()
      ;(0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_17__.scrollToCoords)(this, doc.scrollLeft, doc.scrollTop)
      this.curOp.forceScroll = true
      ;(0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_21__.signalLater)(this, "swapDoc", this, old)
      return old
    }),

    phrase: function(phraseText) {
      let phrases = this.options.phrases
      return phrases && Object.prototype.hasOwnProperty.call(phrases, phraseText) ? phrases[phraseText] : phraseText
    },

    getInputField: function(){return this.display.input.getField()},
    getWrapperElement: function(){return this.display.wrapper},
    getScrollerElement: function(){return this.display.scroller},
    getGutterElement: function(){return this.display.gutters}
  }
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.eventMixin)(CodeMirror)

  CodeMirror.registerHelper = function(type, name, value) {
    if (!helpers.hasOwnProperty(type)) helpers[type] = CodeMirror[type] = {_global: []}
    helpers[type][name] = value
  }
  CodeMirror.registerGlobalHelper = function(type, name, predicate, value) {
    CodeMirror.registerHelper(type, name, value)
    helpers[type]._global.push({pred: predicate, val: value})
  }
}

// Used for horizontal relative motion. Dir is -1 or 1 (left or
// right), unit can be "codepoint", "char", "column" (like char, but
// doesn't cross line boundaries), "word" (across next word), or
// "group" (to the start of next group of word or
// non-word-non-whitespace chars). The visually param controls
// whether, in right-to-left text, direction 1 means to move towards
// the next index in the string, or towards the character to the right
// of the current position. The resulting position will have a
// hitSide=true property if it reached the end of the document.
function findPosH(doc, pos, dir, unit, visually) {
  let oldPos = pos
  let origDir = dir
  let lineObj = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_22__.getLine)(doc, pos.line)
  let lineDir = visually && doc.direction == "rtl" ? -dir : dir
  function findNextLine() {
    let l = pos.line + lineDir
    if (l < doc.first || l >= doc.first + doc.size) return false
    pos = new _line_pos_js__WEBPACK_IMPORTED_MODULE_13__.Pos(l, pos.ch, pos.sticky)
    return lineObj = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_22__.getLine)(doc, l)
  }
  function moveOnce(boundToLine) {
    let next
    if (unit == "codepoint") {
      let ch = lineObj.text.charCodeAt(pos.ch + (dir > 0 ? 0 : -1))
      if (isNaN(ch)) {
        next = null
      } else {
        let astral = dir > 0 ? ch >= 0xD800 && ch < 0xDC00 : ch >= 0xDC00 && ch < 0xDFFF
        next = new _line_pos_js__WEBPACK_IMPORTED_MODULE_13__.Pos(pos.line, Math.max(0, Math.min(lineObj.text.length, pos.ch + dir * (astral ? 2 : 1))), -dir)
      }
    } else if (visually) {
      next = (0,_input_movement_js__WEBPACK_IMPORTED_MODULE_11__.moveVisually)(doc.cm, lineObj, pos, dir)
    } else {
      next = (0,_input_movement_js__WEBPACK_IMPORTED_MODULE_11__.moveLogically)(lineObj, pos, dir)
    }
    if (next == null) {
      if (!boundToLine && findNextLine())
        pos = (0,_input_movement_js__WEBPACK_IMPORTED_MODULE_11__.endOfLine)(visually, doc.cm, lineObj, pos.line, lineDir)
      else
        return false
    } else {
      pos = next
    }
    return true
  }

  if (unit == "char" || unit == "codepoint") {
    moveOnce()
  } else if (unit == "column") {
    moveOnce(true)
  } else if (unit == "word" || unit == "group") {
    let sawType = null, group = unit == "group"
    let helper = doc.cm && doc.cm.getHelper(pos, "wordChars")
    for (let first = true;; first = false) {
      if (dir < 0 && !moveOnce(!first)) break
      let cur = lineObj.text.charAt(pos.ch) || "\n"
      let type = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_20__.isWordChar)(cur, helper) ? "w"
        : group && cur == "\n" ? "n"
        : !group || /\s/.test(cur) ? null
        : "p"
      if (group && !first && !type) type = "s"
      if (sawType && sawType != type) {
        if (dir < 0) {dir = 1; moveOnce(); pos.sticky = "after"}
        break
      }

      if (type) sawType = type
      if (dir > 0 && !moveOnce(!first)) break
    }
  }
  let result = (0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_16__.skipAtomic)(doc, pos, oldPos, origDir, true)
  if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_13__.equalCursorPos)(oldPos, result)) result.hitSide = true
  return result
}

// For relative vertical movement. Dir may be -1 or 1. Unit can be
// "page" or "line". The resulting position will have a hitSide=true
// property if it reached the end of the document.
function findPosV(cm, pos, dir, unit) {
  let doc = cm.doc, x = pos.left, y
  if (unit == "page") {
    let pageSize = Math.min(cm.display.wrapper.clientHeight, window.innerHeight || document.documentElement.clientHeight)
    let moveAmount = Math.max(pageSize - .5 * (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.textHeight)(cm.display), 3)
    y = (dir > 0 ? pos.bottom : pos.top) + dir * moveAmount

  } else if (unit == "line") {
    y = dir > 0 ? pos.bottom + 3 : pos.top - 3
  }
  let target
  for (;;) {
    target = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_14__.coordsChar)(cm, x, y)
    if (!target.outside) break
    if (dir < 0 ? y <= 0 : y >= doc.height) { target.hitSide = true; break }
    y += dir * 5
  }
  return target
}


/***/ }),

/***/ "./node_modules/codemirror/src/edit/mouse_events.js":
/*!**********************************************************!*\
  !*** ./node_modules/codemirror/src/edit/mouse_events.js ***!
  \**********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "onMouseDown": () => (/* binding */ onMouseDown),
/* harmony export */   "clickInGutter": () => (/* binding */ clickInGutter),
/* harmony export */   "onContextMenu": () => (/* binding */ onContextMenu)
/* harmony export */ });
/* harmony import */ var _display_focus_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../display/focus.js */ "./node_modules/codemirror/src/display/focus.js");
/* harmony import */ var _display_operations_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../display/operations.js */ "./node_modules/codemirror/src/display/operations.js");
/* harmony import */ var _display_update_lines_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../display/update_lines.js */ "./node_modules/codemirror/src/display/update_lines.js");
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _measurement_widgets_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../measurement/widgets.js */ "./node_modules/codemirror/src/measurement/widgets.js");
/* harmony import */ var _model_selection_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../model/selection.js */ "./node_modules/codemirror/src/model/selection.js");
/* harmony import */ var _model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ../model/selection_updates.js */ "./node_modules/codemirror/src/model/selection_updates.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_bidi_js__WEBPACK_IMPORTED_MODULE_10__ = __webpack_require__(/*! ../util/bidi.js */ "./node_modules/codemirror/src/util/bidi.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_11__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_12__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _util_feature_detection_js__WEBPACK_IMPORTED_MODULE_13__ = __webpack_require__(/*! ../util/feature_detection.js */ "./node_modules/codemirror/src/util/feature_detection.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_14__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _input_keymap_js__WEBPACK_IMPORTED_MODULE_15__ = __webpack_require__(/*! ../input/keymap.js */ "./node_modules/codemirror/src/input/keymap.js");
/* harmony import */ var _key_events_js__WEBPACK_IMPORTED_MODULE_16__ = __webpack_require__(/*! ./key_events.js */ "./node_modules/codemirror/src/edit/key_events.js");
/* harmony import */ var _commands_js__WEBPACK_IMPORTED_MODULE_17__ = __webpack_require__(/*! ./commands.js */ "./node_modules/codemirror/src/edit/commands.js");





















const DOUBLECLICK_DELAY = 400

class PastClick {
  constructor(time, pos, button) {
    this.time = time
    this.pos = pos
    this.button = button
  }

  compare(time, pos, button) {
    return this.time + DOUBLECLICK_DELAY > time &&
      (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.cmp)(pos, this.pos) == 0 && button == this.button
  }
}

let lastClick, lastDoubleClick
function clickRepeat(pos, button) {
  let now = +new Date
  if (lastDoubleClick && lastDoubleClick.compare(now, pos, button)) {
    lastClick = lastDoubleClick = null
    return "triple"
  } else if (lastClick && lastClick.compare(now, pos, button)) {
    lastDoubleClick = new PastClick(now, pos, button)
    lastClick = null
    return "double"
  } else {
    lastClick = new PastClick(now, pos, button)
    lastDoubleClick = null
    return "single"
  }
}

// A mouse down can be a single click, double click, triple click,
// start of selection drag, start of text drag, new cursor
// (ctrl-click), rectangle drag (alt-drag), or xwin
// middle-click-paste. Or it might be a click on something we should
// not interfere with, such as a scrollbar or widget.
function onMouseDown(e) {
  let cm = this, display = cm.display
  if ((0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.signalDOMEvent)(cm, e) || display.activeTouch && display.input.supportsTouch()) return
  display.input.ensurePolled()
  display.shift = e.shiftKey

  if ((0,_measurement_widgets_js__WEBPACK_IMPORTED_MODULE_6__.eventInWidget)(display, e)) {
    if (!_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.webkit) {
      // Briefly turn off draggability, to allow widgets to do
      // normal dragging things.
      display.scroller.draggable = false
      setTimeout(() => display.scroller.draggable = true, 100)
    }
    return
  }
  if (clickInGutter(cm, e)) return
  let pos = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_5__.posFromMouse)(cm, e), button = (0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_button)(e), repeat = pos ? clickRepeat(pos, button) : "single"
  window.focus()

  // #3261: make sure, that we're not starting a second selection
  if (button == 1 && cm.state.selectingText)
    cm.state.selectingText(e)

  if (pos && handleMappedButton(cm, button, pos, repeat, e)) return

  if (button == 1) {
    if (pos) leftButtonDown(cm, pos, repeat, e)
    else if ((0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_target)(e) == display.scroller) (0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_preventDefault)(e)
  } else if (button == 2) {
    if (pos) (0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.extendSelection)(cm.doc, pos)
    setTimeout(() => display.input.focus(), 20)
  } else if (button == 3) {
    if (_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.captureRightClick) cm.display.input.onContextMenu(e)
    else (0,_display_focus_js__WEBPACK_IMPORTED_MODULE_0__.delayBlurEvent)(cm)
  }
}

function handleMappedButton(cm, button, pos, repeat, event) {
  let name = "Click"
  if (repeat == "double") name = "Double" + name
  else if (repeat == "triple") name = "Triple" + name
  name = (button == 1 ? "Left" : button == 2 ? "Middle" : "Right") + name

  return (0,_key_events_js__WEBPACK_IMPORTED_MODULE_16__.dispatchKey)(cm,  (0,_input_keymap_js__WEBPACK_IMPORTED_MODULE_15__.addModifierNames)(name, event), event, bound => {
    if (typeof bound == "string") bound = _commands_js__WEBPACK_IMPORTED_MODULE_17__.commands[bound]
    if (!bound) return false
    let done = false
    try {
      if (cm.isReadOnly()) cm.state.suppressEdits = true
      done = bound(cm, pos) != _util_misc_js__WEBPACK_IMPORTED_MODULE_14__.Pass
    } finally {
      cm.state.suppressEdits = false
    }
    return done
  })
}

function configureMouse(cm, repeat, event) {
  let option = cm.getOption("configureMouse")
  let value = option ? option(cm, repeat, event) : {}
  if (value.unit == null) {
    let rect = _util_browser_js__WEBPACK_IMPORTED_MODULE_9__.chromeOS ? event.shiftKey && event.metaKey : event.altKey
    value.unit = rect ? "rectangle" : repeat == "single" ? "char" : repeat == "double" ? "word" : "line"
  }
  if (value.extend == null || cm.doc.extend) value.extend = cm.doc.extend || event.shiftKey
  if (value.addNew == null) value.addNew = _util_browser_js__WEBPACK_IMPORTED_MODULE_9__.mac ? event.metaKey : event.ctrlKey
  if (value.moveOnDrag == null) value.moveOnDrag = !(_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.mac ? event.altKey : event.ctrlKey)
  return value
}

function leftButtonDown(cm, pos, repeat, event) {
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.ie) setTimeout((0,_util_misc_js__WEBPACK_IMPORTED_MODULE_14__.bind)(_display_focus_js__WEBPACK_IMPORTED_MODULE_0__.ensureFocus, cm), 0)
  else cm.curOp.focus = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_11__.activeElt)()

  let behavior = configureMouse(cm, repeat, event)

  let sel = cm.doc.sel, contained
  if (cm.options.dragDrop && _util_feature_detection_js__WEBPACK_IMPORTED_MODULE_13__.dragAndDrop && !cm.isReadOnly() &&
      repeat == "single" && (contained = sel.contains(pos)) > -1 &&
      ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.cmp)((contained = sel.ranges[contained]).from(), pos) < 0 || pos.xRel > 0) &&
      ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.cmp)(contained.to(), pos) > 0 || pos.xRel < 0))
    leftButtonStartDrag(cm, event, pos, behavior)
  else
    leftButtonSelect(cm, event, pos, behavior)
}

// Start a text drag. When it ends, see if any dragging actually
// happen, and treat as a click if it didn't.
function leftButtonStartDrag(cm, event, pos, behavior) {
  let display = cm.display, moved = false
  let dragEnd = (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_1__.operation)(cm, e => {
    if (_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.webkit) display.scroller.draggable = false
    cm.state.draggingText = false
    if (cm.state.delayingBlurEvent) {
      if (cm.hasFocus()) cm.state.delayingBlurEvent = false
      else (0,_display_focus_js__WEBPACK_IMPORTED_MODULE_0__.delayBlurEvent)(cm)
    }
    (0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.off)(display.wrapper.ownerDocument, "mouseup", dragEnd)
    ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.off)(display.wrapper.ownerDocument, "mousemove", mouseMove)
    ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.off)(display.scroller, "dragstart", dragStart)
    ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.off)(display.scroller, "drop", dragEnd)
    if (!moved) {
      (0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_preventDefault)(e)
      if (!behavior.addNew)
        (0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.extendSelection)(cm.doc, pos, null, null, behavior.extend)
      // Work around unexplainable focus problem in IE9 (#2127) and Chrome (#3081)
      if ((_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.webkit && !_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.safari) || _util_browser_js__WEBPACK_IMPORTED_MODULE_9__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_9__.ie_version == 9)
        setTimeout(() => {display.wrapper.ownerDocument.body.focus({preventScroll: true}); display.input.focus()}, 20)
      else
        display.input.focus()
    }
  })
  let mouseMove = function(e2) {
    moved = moved || Math.abs(event.clientX - e2.clientX) + Math.abs(event.clientY - e2.clientY) >= 10
  }
  let dragStart = () => moved = true
  // Let the drag handler handle this.
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.webkit) display.scroller.draggable = true
  cm.state.draggingText = dragEnd
  dragEnd.copy = !behavior.moveOnDrag
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.on)(display.wrapper.ownerDocument, "mouseup", dragEnd)
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.on)(display.wrapper.ownerDocument, "mousemove", mouseMove)
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.on)(display.scroller, "dragstart", dragStart)
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.on)(display.scroller, "drop", dragEnd)

  cm.state.delayingBlurEvent = true
  setTimeout(() => display.input.focus(), 20)
  // IE's approach to draggable
  if (display.scroller.dragDrop) display.scroller.dragDrop()
}

function rangeForUnit(cm, pos, unit) {
  if (unit == "char") return new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(pos, pos)
  if (unit == "word") return cm.findWordAt(pos)
  if (unit == "line") return new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.Pos)(pos.line, 0), (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.clipPos)(cm.doc, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.Pos)(pos.line + 1, 0)))
  let result = unit(cm, pos)
  return new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(result.from, result.to)
}

// Normal selection, as opposed to text dragging.
function leftButtonSelect(cm, event, start, behavior) {
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.ie) (0,_display_focus_js__WEBPACK_IMPORTED_MODULE_0__.delayBlurEvent)(cm)
  let display = cm.display, doc = cm.doc
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_preventDefault)(event)

  let ourRange, ourIndex, startSel = doc.sel, ranges = startSel.ranges
  if (behavior.addNew && !behavior.extend) {
    ourIndex = doc.sel.contains(start)
    if (ourIndex > -1)
      ourRange = ranges[ourIndex]
    else
      ourRange = new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(start, start)
  } else {
    ourRange = doc.sel.primary()
    ourIndex = doc.sel.primIndex
  }

  if (behavior.unit == "rectangle") {
    if (!behavior.addNew) ourRange = new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(start, start)
    start = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_5__.posFromMouse)(cm, event, true, true)
    ourIndex = -1
  } else {
    let range = rangeForUnit(cm, start, behavior.unit)
    if (behavior.extend)
      ourRange = (0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.extendRange)(ourRange, range.anchor, range.head, behavior.extend)
    else
      ourRange = range
  }

  if (!behavior.addNew) {
    ourIndex = 0
    ;(0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.setSelection)(doc, new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Selection([ourRange], 0), _util_misc_js__WEBPACK_IMPORTED_MODULE_14__.sel_mouse)
    startSel = doc.sel
  } else if (ourIndex == -1) {
    ourIndex = ranges.length
    ;(0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.setSelection)(doc, (0,_model_selection_js__WEBPACK_IMPORTED_MODULE_7__.normalizeSelection)(cm, ranges.concat([ourRange]), ourIndex),
                 {scroll: false, origin: "*mouse"})
  } else if (ranges.length > 1 && ranges[ourIndex].empty() && behavior.unit == "char" && !behavior.extend) {
    (0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.setSelection)(doc, (0,_model_selection_js__WEBPACK_IMPORTED_MODULE_7__.normalizeSelection)(cm, ranges.slice(0, ourIndex).concat(ranges.slice(ourIndex + 1)), 0),
                 {scroll: false, origin: "*mouse"})
    startSel = doc.sel
  } else {
    (0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.replaceOneSelection)(doc, ourIndex, ourRange, _util_misc_js__WEBPACK_IMPORTED_MODULE_14__.sel_mouse)
  }

  let lastPos = start
  function extendTo(pos) {
    if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.cmp)(lastPos, pos) == 0) return
    lastPos = pos

    if (behavior.unit == "rectangle") {
      let ranges = [], tabSize = cm.options.tabSize
      let startCol = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_14__.countColumn)((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_4__.getLine)(doc, start.line).text, start.ch, tabSize)
      let posCol = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_14__.countColumn)((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_4__.getLine)(doc, pos.line).text, pos.ch, tabSize)
      let left = Math.min(startCol, posCol), right = Math.max(startCol, posCol)
      for (let line = Math.min(start.line, pos.line), end = Math.min(cm.lastLine(), Math.max(start.line, pos.line));
           line <= end; line++) {
        let text = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_4__.getLine)(doc, line).text, leftPos = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_14__.findColumn)(text, left, tabSize)
        if (left == right)
          ranges.push(new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.Pos)(line, leftPos), (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.Pos)(line, leftPos)))
        else if (text.length > leftPos)
          ranges.push(new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.Pos)(line, leftPos), (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.Pos)(line, (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_14__.findColumn)(text, right, tabSize))))
      }
      if (!ranges.length) ranges.push(new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(start, start))
      ;(0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.setSelection)(doc, (0,_model_selection_js__WEBPACK_IMPORTED_MODULE_7__.normalizeSelection)(cm, startSel.ranges.slice(0, ourIndex).concat(ranges), ourIndex),
                   {origin: "*mouse", scroll: false})
      cm.scrollIntoView(pos)
    } else {
      let oldRange = ourRange
      let range = rangeForUnit(cm, pos, behavior.unit)
      let anchor = oldRange.anchor, head
      if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.cmp)(range.anchor, anchor) > 0) {
        head = range.head
        anchor = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.minPos)(oldRange.from(), range.anchor)
      } else {
        head = range.anchor
        anchor = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.maxPos)(oldRange.to(), range.head)
      }
      let ranges = startSel.ranges.slice(0)
      ranges[ourIndex] = bidiSimplify(cm, new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.clipPos)(doc, anchor), head))
      ;(0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_8__.setSelection)(doc, (0,_model_selection_js__WEBPACK_IMPORTED_MODULE_7__.normalizeSelection)(cm, ranges, ourIndex), _util_misc_js__WEBPACK_IMPORTED_MODULE_14__.sel_mouse)
    }
  }

  let editorSize = display.wrapper.getBoundingClientRect()
  // Used to ensure timeout re-tries don't fire when another extend
  // happened in the meantime (clearTimeout isn't reliable -- at
  // least on Chrome, the timeouts still happen even when cleared,
  // if the clear happens after their scheduled firing time).
  let counter = 0

  function extend(e) {
    let curCount = ++counter
    let cur = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_5__.posFromMouse)(cm, e, true, behavior.unit == "rectangle")
    if (!cur) return
    if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.cmp)(cur, lastPos) != 0) {
      cm.curOp.focus = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_11__.activeElt)()
      extendTo(cur)
      let visible = (0,_display_update_lines_js__WEBPACK_IMPORTED_MODULE_2__.visibleLines)(display, doc)
      if (cur.line >= visible.to || cur.line < visible.from)
        setTimeout((0,_display_operations_js__WEBPACK_IMPORTED_MODULE_1__.operation)(cm, () => {if (counter == curCount) extend(e)}), 150)
    } else {
      let outside = e.clientY < editorSize.top ? -20 : e.clientY > editorSize.bottom ? 20 : 0
      if (outside) setTimeout((0,_display_operations_js__WEBPACK_IMPORTED_MODULE_1__.operation)(cm, () => {
        if (counter != curCount) return
        display.scroller.scrollTop += outside
        extend(e)
      }), 50)
    }
  }

  function done(e) {
    cm.state.selectingText = false
    counter = Infinity
    // If e is null or undefined we interpret this as someone trying
    // to explicitly cancel the selection rather than the user
    // letting go of the mouse button.
    if (e) {
      (0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_preventDefault)(e)
      display.input.focus()
    }
    (0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.off)(display.wrapper.ownerDocument, "mousemove", move)
    ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.off)(display.wrapper.ownerDocument, "mouseup", up)
    doc.history.lastSelOrigin = null
  }

  let move = (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_1__.operation)(cm, e => {
    if (e.buttons === 0 || !(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_button)(e)) done(e)
    else extend(e)
  })
  let up = (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_1__.operation)(cm, done)
  cm.state.selectingText = up
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.on)(display.wrapper.ownerDocument, "mousemove", move)
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.on)(display.wrapper.ownerDocument, "mouseup", up)
}

// Used when mouse-selecting to adjust the anchor to the proper side
// of a bidi jump depending on the visual position of the head.
function bidiSimplify(cm, range) {
  let {anchor, head} = range, anchorLine = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_4__.getLine)(cm.doc, anchor.line)
  if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_3__.cmp)(anchor, head) == 0 && anchor.sticky == head.sticky) return range
  let order = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_10__.getOrder)(anchorLine)
  if (!order) return range
  let index = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_10__.getBidiPartAt)(order, anchor.ch, anchor.sticky), part = order[index]
  if (part.from != anchor.ch && part.to != anchor.ch) return range
  let boundary = index + ((part.from == anchor.ch) == (part.level != 1) ? 0 : 1)
  if (boundary == 0 || boundary == order.length) return range

  // Compute the relative visual position of the head compared to the
  // anchor (<0 is to the left, >0 to the right)
  let leftSide
  if (head.line != anchor.line) {
    leftSide = (head.line - anchor.line) * (cm.doc.direction == "ltr" ? 1 : -1) > 0
  } else {
    let headIndex = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_10__.getBidiPartAt)(order, head.ch, head.sticky)
    let dir = headIndex - index || (head.ch - anchor.ch) * (part.level == 1 ? -1 : 1)
    if (headIndex == boundary - 1 || headIndex == boundary)
      leftSide = dir < 0
    else
      leftSide = dir > 0
  }

  let usePart = order[boundary + (leftSide ? -1 : 0)]
  let from = leftSide == (usePart.level == 1)
  let ch = from ? usePart.from : usePart.to, sticky = from ? "after" : "before"
  return anchor.ch == ch && anchor.sticky == sticky ? range : new _model_selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(new _line_pos_js__WEBPACK_IMPORTED_MODULE_3__.Pos(anchor.line, ch, sticky), head)
}


// Determines whether an event happened in the gutter, and fires the
// handlers for the corresponding event.
function gutterEvent(cm, e, type, prevent) {
  let mX, mY
  if (e.touches) {
    mX = e.touches[0].clientX
    mY = e.touches[0].clientY
  } else {
    try { mX = e.clientX; mY = e.clientY }
    catch(e) { return false }
  }
  if (mX >= Math.floor(cm.display.gutters.getBoundingClientRect().right)) return false
  if (prevent) (0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_preventDefault)(e)

  let display = cm.display
  let lineBox = display.lineDiv.getBoundingClientRect()

  if (mY > lineBox.bottom || !(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.hasHandler)(cm, type)) return (0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_defaultPrevented)(e)
  mY -= lineBox.top - display.viewOffset

  for (let i = 0; i < cm.display.gutterSpecs.length; ++i) {
    let g = display.gutters.childNodes[i]
    if (g && g.getBoundingClientRect().right >= mX) {
      let line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_4__.lineAtHeight)(cm.doc, mY)
      let gutter = cm.display.gutterSpecs[i]
      ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.signal)(cm, type, cm, line, gutter.className, e)
      return (0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.e_defaultPrevented)(e)
    }
  }
}

function clickInGutter(cm, e) {
  return gutterEvent(cm, e, "gutterClick", true)
}

// CONTEXT MENU HANDLING

// To make the context menu work, we need to briefly unhide the
// textarea (making it as unobtrusive as possible) to let the
// right-click take effect on it.
function onContextMenu(cm, e) {
  if ((0,_measurement_widgets_js__WEBPACK_IMPORTED_MODULE_6__.eventInWidget)(cm.display, e) || contextMenuInGutter(cm, e)) return
  if ((0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.signalDOMEvent)(cm, e, "contextmenu")) return
  if (!_util_browser_js__WEBPACK_IMPORTED_MODULE_9__.captureRightClick) cm.display.input.onContextMenu(e)
}

function contextMenuInGutter(cm, e) {
  if (!(0,_util_event_js__WEBPACK_IMPORTED_MODULE_12__.hasHandler)(cm, "gutterContextMenu")) return false
  return gutterEvent(cm, e, "gutterContextMenu", false)
}


/***/ }),

/***/ "./node_modules/codemirror/src/input/indent.js":
/*!*****************************************************!*\
  !*** ./node_modules/codemirror/src/input/indent.js ***!
  \*****************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "indentLine": () => (/* binding */ indentLine)
/* harmony export */ });
/* harmony import */ var _line_highlight_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/highlight.js */ "./node_modules/codemirror/src/line/highlight.js");
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _model_changes_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../model/changes.js */ "./node_modules/codemirror/src/model/changes.js");
/* harmony import */ var _model_selection_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../model/selection.js */ "./node_modules/codemirror/src/model/selection.js");
/* harmony import */ var _model_selection_updates_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../model/selection_updates.js */ "./node_modules/codemirror/src/model/selection_updates.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");








// Indent the given line. The how parameter can be "smart",
// "add"/null, "subtract", or "prev". When aggressive is false
// (typically set to true for forced single-line indents), empty
// lines are not indented, and places where the mode returns Pass
// are left alone.
function indentLine(cm, n, how, aggressive) {
  let doc = cm.doc, state
  if (how == null) how = "add"
  if (how == "smart") {
    // Fall back to "prev" when the mode doesn't have an indentation
    // method.
    if (!doc.mode.indent) how = "prev"
    else state = (0,_line_highlight_js__WEBPACK_IMPORTED_MODULE_0__.getContextBefore)(cm, n).state
  }

  let tabSize = cm.options.tabSize
  let line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__.getLine)(doc, n), curSpace = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_6__.countColumn)(line.text, null, tabSize)
  if (line.stateAfter) line.stateAfter = null
  let curSpaceString = line.text.match(/^\s*/)[0], indentation
  if (!aggressive && !/\S/.test(line.text)) {
    indentation = 0
    how = "not"
  } else if (how == "smart") {
    indentation = doc.mode.indent(state, line.text.slice(curSpaceString.length), line.text)
    if (indentation == _util_misc_js__WEBPACK_IMPORTED_MODULE_6__.Pass || indentation > 150) {
      if (!aggressive) return
      how = "prev"
    }
  }
  if (how == "prev") {
    if (n > doc.first) indentation = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_6__.countColumn)((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__.getLine)(doc, n-1).text, null, tabSize)
    else indentation = 0
  } else if (how == "add") {
    indentation = curSpace + cm.options.indentUnit
  } else if (how == "subtract") {
    indentation = curSpace - cm.options.indentUnit
  } else if (typeof how == "number") {
    indentation = curSpace + how
  }
  indentation = Math.max(0, indentation)

  let indentString = "", pos = 0
  if (cm.options.indentWithTabs)
    for (let i = Math.floor(indentation / tabSize); i; --i) {pos += tabSize; indentString += "\t"}
  if (pos < indentation) indentString += (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_6__.spaceStr)(indentation - pos)

  if (indentString != curSpaceString) {
    (0,_model_changes_js__WEBPACK_IMPORTED_MODULE_3__.replaceRange)(doc, indentString, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_1__.Pos)(n, 0), (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_1__.Pos)(n, curSpaceString.length), "+input")
    line.stateAfter = null
    return true
  } else {
    // Ensure that, if the cursor was in the whitespace at the start
    // of the line, it is moved to the end of that space.
    for (let i = 0; i < doc.sel.ranges.length; i++) {
      let range = doc.sel.ranges[i]
      if (range.head.line == n && range.head.ch < curSpaceString.length) {
        let pos = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_1__.Pos)(n, curSpaceString.length)
        ;(0,_model_selection_updates_js__WEBPACK_IMPORTED_MODULE_5__.replaceOneSelection)(doc, i, new _model_selection_js__WEBPACK_IMPORTED_MODULE_4__.Range(pos, pos))
        break
      }
    }
  }
}


/***/ }),

/***/ "./node_modules/codemirror/src/input/input.js":
/*!****************************************************!*\
  !*** ./node_modules/codemirror/src/input/input.js ***!
  \****************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "lastCopied": () => (/* binding */ lastCopied),
/* harmony export */   "setLastCopied": () => (/* binding */ setLastCopied),
/* harmony export */   "applyTextInput": () => (/* binding */ applyTextInput),
/* harmony export */   "handlePaste": () => (/* binding */ handlePaste),
/* harmony export */   "triggerElectric": () => (/* binding */ triggerElectric),
/* harmony export */   "copyableRanges": () => (/* binding */ copyableRanges),
/* harmony export */   "disableBrowserMagic": () => (/* binding */ disableBrowserMagic),
/* harmony export */   "hiddenTextarea": () => (/* binding */ hiddenTextarea)
/* harmony export */ });
/* harmony import */ var _display_operations_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../display/operations.js */ "./node_modules/codemirror/src/display/operations.js");
/* harmony import */ var _display_scrolling_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../display/scrolling.js */ "./node_modules/codemirror/src/display/scrolling.js");
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _model_changes_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../model/changes.js */ "./node_modules/codemirror/src/model/changes.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _util_operation_group_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ../util/operation_group.js */ "./node_modules/codemirror/src/util/operation_group.js");
/* harmony import */ var _util_feature_detection_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ../util/feature_detection.js */ "./node_modules/codemirror/src/util/feature_detection.js");
/* harmony import */ var _indent_js__WEBPACK_IMPORTED_MODULE_10__ = __webpack_require__(/*! ./indent.js */ "./node_modules/codemirror/src/input/indent.js");













// This will be set to a {lineWise: bool, text: [string]} object, so
// that, when pasting, we know what kind of selections the copied
// text was made out of.
let lastCopied = null

function setLastCopied(newLastCopied) {
  lastCopied = newLastCopied
}

function applyTextInput(cm, inserted, deleted, sel, origin) {
  let doc = cm.doc
  cm.display.shift = false
  if (!sel) sel = doc.sel

  let recent = +new Date - 200
  let paste = origin == "paste" || cm.state.pasteIncoming > recent
  let textLines = (0,_util_feature_detection_js__WEBPACK_IMPORTED_MODULE_9__.splitLinesAuto)(inserted), multiPaste = null
  // When pasting N lines into N selections, insert one line per selection
  if (paste && sel.ranges.length > 1) {
    if (lastCopied && lastCopied.text.join("\n") == inserted) {
      if (sel.ranges.length % lastCopied.text.length == 0) {
        multiPaste = []
        for (let i = 0; i < lastCopied.text.length; i++)
          multiPaste.push(doc.splitLines(lastCopied.text[i]))
      }
    } else if (textLines.length == sel.ranges.length && cm.options.pasteLinesPerSelection) {
      multiPaste = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_7__.map)(textLines, l => [l])
    }
  }

  let updateInput = cm.curOp.updateInput
  // Normal behavior is to insert the new text into every selection
  for (let i = sel.ranges.length - 1; i >= 0; i--) {
    let range = sel.ranges[i]
    let from = range.from(), to = range.to()
    if (range.empty()) {
      if (deleted && deleted > 0) // Handle deletion
        from = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(from.line, from.ch - deleted)
      else if (cm.state.overwrite && !paste) // Handle overwrite
        to = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(to.line, Math.min((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, to.line).text.length, to.ch + (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_7__.lst)(textLines).length))
      else if (paste && lastCopied && lastCopied.lineWise && lastCopied.text.join("\n") == textLines.join("\n"))
        from = to = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(from.line, 0)
    }
    let changeEvent = {from: from, to: to, text: multiPaste ? multiPaste[i % multiPaste.length] : textLines,
                       origin: origin || (paste ? "paste" : cm.state.cutIncoming > recent ? "cut" : "+input")}
    ;(0,_model_changes_js__WEBPACK_IMPORTED_MODULE_4__.makeChange)(cm.doc, changeEvent)
    ;(0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_8__.signalLater)(cm, "inputRead", cm, changeEvent)
  }
  if (inserted && !paste)
    triggerElectric(cm, inserted)

  ;(0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_1__.ensureCursorVisible)(cm)
  if (cm.curOp.updateInput < 2) cm.curOp.updateInput = updateInput
  cm.curOp.typing = true
  cm.state.pasteIncoming = cm.state.cutIncoming = -1
}

function handlePaste(e, cm) {
  let pasted = e.clipboardData && e.clipboardData.getData("Text")
  if (pasted) {
    e.preventDefault()
    if (!cm.isReadOnly() && !cm.options.disableInput)
      (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_0__.runInOp)(cm, () => applyTextInput(cm, pasted, 0, null, "paste"))
    return true
  }
}

function triggerElectric(cm, inserted) {
  // When an 'electric' character is inserted, immediately trigger a reindent
  if (!cm.options.electricChars || !cm.options.smartIndent) return
  let sel = cm.doc.sel

  for (let i = sel.ranges.length - 1; i >= 0; i--) {
    let range = sel.ranges[i]
    if (range.head.ch > 100 || (i && sel.ranges[i - 1].head.line == range.head.line)) continue
    let mode = cm.getModeAt(range.head)
    let indented = false
    if (mode.electricChars) {
      for (let j = 0; j < mode.electricChars.length; j++)
        if (inserted.indexOf(mode.electricChars.charAt(j)) > -1) {
          indented = (0,_indent_js__WEBPACK_IMPORTED_MODULE_10__.indentLine)(cm, range.head.line, "smart")
          break
        }
    } else if (mode.electricInput) {
      if (mode.electricInput.test((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(cm.doc, range.head.line).text.slice(0, range.head.ch)))
        indented = (0,_indent_js__WEBPACK_IMPORTED_MODULE_10__.indentLine)(cm, range.head.line, "smart")
    }
    if (indented) (0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_8__.signalLater)(cm, "electricInput", cm, range.head.line)
  }
}

function copyableRanges(cm) {
  let text = [], ranges = []
  for (let i = 0; i < cm.doc.sel.ranges.length; i++) {
    let line = cm.doc.sel.ranges[i].head.line
    let lineRange = {anchor: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(line, 0), head: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(line + 1, 0)}
    ranges.push(lineRange)
    text.push(cm.getRange(lineRange.anchor, lineRange.head))
  }
  return {text: text, ranges: ranges}
}

function disableBrowserMagic(field, spellcheck, autocorrect, autocapitalize) {
  field.setAttribute("autocorrect", autocorrect ? "" : "off")
  field.setAttribute("autocapitalize", autocapitalize ? "" : "off")
  field.setAttribute("spellcheck", !!spellcheck)
}

function hiddenTextarea() {
  let te = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.elt)("textarea", null, null, "position: absolute; bottom: -1em; padding: 0; width: 1px; height: 1em; outline: none")
  let div = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.elt)("div", [te], null, "overflow: hidden; position: relative; width: 3px; height: 0px;")
  // The textarea is kept positioned near the cursor to prevent the
  // fact that it'll be scrolled into view on input from scrolling
  // our fake cursor out of view. On webkit, when wrap=off, paste is
  // very slow. So make the area wide instead.
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_5__.webkit) te.style.width = "1000px"
  else te.setAttribute("wrap", "off")
  // If border: 0; -- iOS fails to open keyboard (issue #1287)
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_5__.ios) te.style.border = "1px solid black"
  disableBrowserMagic(te)
  return div
}


/***/ }),

/***/ "./node_modules/codemirror/src/input/keymap.js":
/*!*****************************************************!*\
  !*** ./node_modules/codemirror/src/input/keymap.js ***!
  \*****************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "keyMap": () => (/* binding */ keyMap),
/* harmony export */   "normalizeKeyMap": () => (/* binding */ normalizeKeyMap),
/* harmony export */   "lookupKey": () => (/* binding */ lookupKey),
/* harmony export */   "isModifierKey": () => (/* binding */ isModifierKey),
/* harmony export */   "addModifierNames": () => (/* binding */ addModifierNames),
/* harmony export */   "keyName": () => (/* binding */ keyName),
/* harmony export */   "getKeyMap": () => (/* binding */ getKeyMap)
/* harmony export */ });
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _keynames_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ./keynames.js */ "./node_modules/codemirror/src/input/keynames.js");





let keyMap = {}

keyMap.basic = {
  "Left": "goCharLeft", "Right": "goCharRight", "Up": "goLineUp", "Down": "goLineDown",
  "End": "goLineEnd", "Home": "goLineStartSmart", "PageUp": "goPageUp", "PageDown": "goPageDown",
  "Delete": "delCharAfter", "Backspace": "delCharBefore", "Shift-Backspace": "delCharBefore",
  "Tab": "defaultTab", "Shift-Tab": "indentAuto",
  "Enter": "newlineAndIndent", "Insert": "toggleOverwrite",
  "Esc": "singleSelection"
}
// Note that the save and find-related commands aren't defined by
// default. User code or addons can define them. Unknown commands
// are simply ignored.
keyMap.pcDefault = {
  "Ctrl-A": "selectAll", "Ctrl-D": "deleteLine", "Ctrl-Z": "undo", "Shift-Ctrl-Z": "redo", "Ctrl-Y": "redo",
  "Ctrl-Home": "goDocStart", "Ctrl-End": "goDocEnd", "Ctrl-Up": "goLineUp", "Ctrl-Down": "goLineDown",
  "Ctrl-Left": "goGroupLeft", "Ctrl-Right": "goGroupRight", "Alt-Left": "goLineStart", "Alt-Right": "goLineEnd",
  "Ctrl-Backspace": "delGroupBefore", "Ctrl-Delete": "delGroupAfter", "Ctrl-S": "save", "Ctrl-F": "find",
  "Ctrl-G": "findNext", "Shift-Ctrl-G": "findPrev", "Shift-Ctrl-F": "replace", "Shift-Ctrl-R": "replaceAll",
  "Ctrl-[": "indentLess", "Ctrl-]": "indentMore",
  "Ctrl-U": "undoSelection", "Shift-Ctrl-U": "redoSelection", "Alt-U": "redoSelection",
  "fallthrough": "basic"
}
// Very basic readline/emacs-style bindings, which are standard on Mac.
keyMap.emacsy = {
  "Ctrl-F": "goCharRight", "Ctrl-B": "goCharLeft", "Ctrl-P": "goLineUp", "Ctrl-N": "goLineDown",
  "Ctrl-A": "goLineStart", "Ctrl-E": "goLineEnd", "Ctrl-V": "goPageDown", "Shift-Ctrl-V": "goPageUp",
  "Ctrl-D": "delCharAfter", "Ctrl-H": "delCharBefore", "Alt-Backspace": "delWordBefore", "Ctrl-K": "killLine",
  "Ctrl-T": "transposeChars", "Ctrl-O": "openLine"
}
keyMap.macDefault = {
  "Cmd-A": "selectAll", "Cmd-D": "deleteLine", "Cmd-Z": "undo", "Shift-Cmd-Z": "redo", "Cmd-Y": "redo",
  "Cmd-Home": "goDocStart", "Cmd-Up": "goDocStart", "Cmd-End": "goDocEnd", "Cmd-Down": "goDocEnd", "Alt-Left": "goGroupLeft",
  "Alt-Right": "goGroupRight", "Cmd-Left": "goLineLeft", "Cmd-Right": "goLineRight", "Alt-Backspace": "delGroupBefore",
  "Ctrl-Alt-Backspace": "delGroupAfter", "Alt-Delete": "delGroupAfter", "Cmd-S": "save", "Cmd-F": "find",
  "Cmd-G": "findNext", "Shift-Cmd-G": "findPrev", "Cmd-Alt-F": "replace", "Shift-Cmd-Alt-F": "replaceAll",
  "Cmd-[": "indentLess", "Cmd-]": "indentMore", "Cmd-Backspace": "delWrappedLineLeft", "Cmd-Delete": "delWrappedLineRight",
  "Cmd-U": "undoSelection", "Shift-Cmd-U": "redoSelection", "Ctrl-Up": "goDocStart", "Ctrl-Down": "goDocEnd",
  "fallthrough": ["basic", "emacsy"]
}
keyMap["default"] = _util_browser_js__WEBPACK_IMPORTED_MODULE_0__.mac ? keyMap.macDefault : keyMap.pcDefault

// KEYMAP DISPATCH

function normalizeKeyName(name) {
  let parts = name.split(/-(?!$)/)
  name = parts[parts.length - 1]
  let alt, ctrl, shift, cmd
  for (let i = 0; i < parts.length - 1; i++) {
    let mod = parts[i]
    if (/^(cmd|meta|m)$/i.test(mod)) cmd = true
    else if (/^a(lt)?$/i.test(mod)) alt = true
    else if (/^(c|ctrl|control)$/i.test(mod)) ctrl = true
    else if (/^s(hift)?$/i.test(mod)) shift = true
    else throw new Error("Unrecognized modifier name: " + mod)
  }
  if (alt) name = "Alt-" + name
  if (ctrl) name = "Ctrl-" + name
  if (cmd) name = "Cmd-" + name
  if (shift) name = "Shift-" + name
  return name
}

// This is a kludge to keep keymaps mostly working as raw objects
// (backwards compatibility) while at the same time support features
// like normalization and multi-stroke key bindings. It compiles a
// new normalized keymap, and then updates the old object to reflect
// this.
function normalizeKeyMap(keymap) {
  let copy = {}
  for (let keyname in keymap) if (keymap.hasOwnProperty(keyname)) {
    let value = keymap[keyname]
    if (/^(name|fallthrough|(de|at)tach)$/.test(keyname)) continue
    if (value == "...") { delete keymap[keyname]; continue }

    let keys = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_1__.map)(keyname.split(" "), normalizeKeyName)
    for (let i = 0; i < keys.length; i++) {
      let val, name
      if (i == keys.length - 1) {
        name = keys.join(" ")
        val = value
      } else {
        name = keys.slice(0, i + 1).join(" ")
        val = "..."
      }
      let prev = copy[name]
      if (!prev) copy[name] = val
      else if (prev != val) throw new Error("Inconsistent bindings for " + name)
    }
    delete keymap[keyname]
  }
  for (let prop in copy) keymap[prop] = copy[prop]
  return keymap
}

function lookupKey(key, map, handle, context) {
  map = getKeyMap(map)
  let found = map.call ? map.call(key, context) : map[key]
  if (found === false) return "nothing"
  if (found === "...") return "multi"
  if (found != null && handle(found)) return "handled"

  if (map.fallthrough) {
    if (Object.prototype.toString.call(map.fallthrough) != "[object Array]")
      return lookupKey(key, map.fallthrough, handle, context)
    for (let i = 0; i < map.fallthrough.length; i++) {
      let result = lookupKey(key, map.fallthrough[i], handle, context)
      if (result) return result
    }
  }
}

// Modifier key presses don't count as 'real' key presses for the
// purpose of keymap fallthrough.
function isModifierKey(value) {
  let name = typeof value == "string" ? value : _keynames_js__WEBPACK_IMPORTED_MODULE_2__.keyNames[value.keyCode]
  return name == "Ctrl" || name == "Alt" || name == "Shift" || name == "Mod"
}

function addModifierNames(name, event, noShift) {
  let base = name
  if (event.altKey && base != "Alt") name = "Alt-" + name
  if ((_util_browser_js__WEBPACK_IMPORTED_MODULE_0__.flipCtrlCmd ? event.metaKey : event.ctrlKey) && base != "Ctrl") name = "Ctrl-" + name
  if ((_util_browser_js__WEBPACK_IMPORTED_MODULE_0__.flipCtrlCmd ? event.ctrlKey : event.metaKey) && base != "Mod") name = "Cmd-" + name
  if (!noShift && event.shiftKey && base != "Shift") name = "Shift-" + name
  return name
}

// Look up the name of a key as indicated by an event object.
function keyName(event, noShift) {
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_0__.presto && event.keyCode == 34 && event["char"]) return false
  let name = _keynames_js__WEBPACK_IMPORTED_MODULE_2__.keyNames[event.keyCode]
  if (name == null || event.altGraphKey) return false
  // Ctrl-ScrollLock has keyCode 3, same as Ctrl-Pause,
  // so we'll use event.code when available (Chrome 48+, FF 38+, Safari 10.1+)
  if (event.keyCode == 3 && event.code) name = event.code
  return addModifierNames(name, event, noShift)
}

function getKeyMap(val) {
  return typeof val == "string" ? keyMap[val] : val
}


/***/ }),

/***/ "./node_modules/codemirror/src/input/keynames.js":
/*!*******************************************************!*\
  !*** ./node_modules/codemirror/src/input/keynames.js ***!
  \*******************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "keyNames": () => (/* binding */ keyNames)
/* harmony export */ });
let keyNames = {
  3: "Pause", 8: "Backspace", 9: "Tab", 13: "Enter", 16: "Shift", 17: "Ctrl", 18: "Alt",
  19: "Pause", 20: "CapsLock", 27: "Esc", 32: "Space", 33: "PageUp", 34: "PageDown", 35: "End",
  36: "Home", 37: "Left", 38: "Up", 39: "Right", 40: "Down", 44: "PrintScrn", 45: "Insert",
  46: "Delete", 59: ";", 61: "=", 91: "Mod", 92: "Mod", 93: "Mod",
  106: "*", 107: "=", 109: "-", 110: ".", 111: "/", 145: "ScrollLock",
  173: "-", 186: ";", 187: "=", 188: ",", 189: "-", 190: ".", 191: "/", 192: "`", 219: "[", 220: "\\",
  221: "]", 222: "'", 224: "Mod", 63232: "Up", 63233: "Down", 63234: "Left", 63235: "Right", 63272: "Delete",
  63273: "Home", 63275: "End", 63276: "PageUp", 63277: "PageDown", 63302: "Insert"
}

// Number keys
for (let i = 0; i < 10; i++) keyNames[i + 48] = keyNames[i + 96] = String(i)
// Alphabetic keys
for (let i = 65; i <= 90; i++) keyNames[i] = String.fromCharCode(i)
// Function keys
for (let i = 1; i <= 12; i++) keyNames[i + 111] = keyNames[i + 63235] = "F" + i


/***/ }),

/***/ "./node_modules/codemirror/src/input/movement.js":
/*!*******************************************************!*\
  !*** ./node_modules/codemirror/src/input/movement.js ***!
  \*******************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "moveLogically": () => (/* binding */ moveLogically),
/* harmony export */   "endOfLine": () => (/* binding */ endOfLine),
/* harmony export */   "moveVisually": () => (/* binding */ moveVisually)
/* harmony export */ });
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_bidi_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../util/bidi.js */ "./node_modules/codemirror/src/util/bidi.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");





function moveCharLogically(line, ch, dir) {
  let target = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_3__.skipExtendingChars)(line.text, ch + dir, dir)
  return target < 0 || target > line.text.length ? null : target
}

function moveLogically(line, start, dir) {
  let ch = moveCharLogically(line, start.ch, dir)
  return ch == null ? null : new _line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos(start.line, ch, dir < 0 ? "after" : "before")
}

function endOfLine(visually, cm, lineObj, lineNo, dir) {
  if (visually) {
    if (cm.doc.direction == "rtl") dir = -dir
    let order = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_2__.getOrder)(lineObj, cm.doc.direction)
    if (order) {
      let part = dir < 0 ? (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_3__.lst)(order) : order[0]
      let moveInStorageOrder = (dir < 0) == (part.level == 1)
      let sticky = moveInStorageOrder ? "after" : "before"
      let ch
      // With a wrapped rtl chunk (possibly spanning multiple bidi parts),
      // it could be that the last bidi part is not on the last visual line,
      // since visual lines contain content order-consecutive chunks.
      // Thus, in rtl, we are looking for the first (content-order) character
      // in the rtl chunk that is on the last line (that is, the same line
      // as the last (content-order) character).
      if (part.level > 0 || cm.doc.direction == "rtl") {
        let prep = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.prepareMeasureForLine)(cm, lineObj)
        ch = dir < 0 ? lineObj.text.length - 1 : 0
        let targetTop = (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.measureCharPrepared)(cm, prep, ch).top
        ch = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_3__.findFirst)(ch => (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.measureCharPrepared)(cm, prep, ch).top == targetTop, (dir < 0) == (part.level == 1) ? part.from : part.to - 1, ch)
        if (sticky == "before") ch = moveCharLogically(lineObj, ch, 1)
      } else ch = dir < 0 ? part.to : part.from
      return new _line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos(lineNo, ch, sticky)
    }
  }
  return new _line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos(lineNo, dir < 0 ? lineObj.text.length : 0, dir < 0 ? "before" : "after")
}

function moveVisually(cm, line, start, dir) {
  let bidi = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_2__.getOrder)(line, cm.doc.direction)
  if (!bidi) return moveLogically(line, start, dir)
  if (start.ch >= line.text.length) {
    start.ch = line.text.length
    start.sticky = "before"
  } else if (start.ch <= 0) {
    start.ch = 0
    start.sticky = "after"
  }
  let partPos = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_2__.getBidiPartAt)(bidi, start.ch, start.sticky), part = bidi[partPos]
  if (cm.doc.direction == "ltr" && part.level % 2 == 0 && (dir > 0 ? part.to > start.ch : part.from < start.ch)) {
    // Case 1: We move within an ltr part in an ltr editor. Even with wrapped lines,
    // nothing interesting happens.
    return moveLogically(line, start, dir)
  }

  let mv = (pos, dir) => moveCharLogically(line, pos instanceof _line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos ? pos.ch : pos, dir)
  let prep
  let getWrappedLineExtent = ch => {
    if (!cm.options.lineWrapping) return {begin: 0, end: line.text.length}
    prep = prep || (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.prepareMeasureForLine)(cm, line)
    return (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_1__.wrappedLineExtentChar)(cm, line, prep, ch)
  }
  let wrappedLineExtent = getWrappedLineExtent(start.sticky == "before" ? mv(start, -1) : start.ch)

  if (cm.doc.direction == "rtl" || part.level == 1) {
    let moveInStorageOrder = (part.level == 1) == (dir < 0)
    let ch = mv(start, moveInStorageOrder ? 1 : -1)
    if (ch != null && (!moveInStorageOrder ? ch >= part.from && ch >= wrappedLineExtent.begin : ch <= part.to && ch <= wrappedLineExtent.end)) {
      // Case 2: We move within an rtl part or in an rtl editor on the same visual line
      let sticky = moveInStorageOrder ? "before" : "after"
      return new _line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos(start.line, ch, sticky)
    }
  }

  // Case 3: Could not move within this bidi part in this visual line, so leave
  // the current bidi part

  let searchInVisualLine = (partPos, dir, wrappedLineExtent) => {
    let getRes = (ch, moveInStorageOrder) => moveInStorageOrder
      ? new _line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos(start.line, mv(ch, 1), "before")
      : new _line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos(start.line, ch, "after")

    for (; partPos >= 0 && partPos < bidi.length; partPos += dir) {
      let part = bidi[partPos]
      let moveInStorageOrder = (dir > 0) == (part.level != 1)
      let ch = moveInStorageOrder ? wrappedLineExtent.begin : mv(wrappedLineExtent.end, -1)
      if (part.from <= ch && ch < part.to) return getRes(ch, moveInStorageOrder)
      ch = moveInStorageOrder ? part.from : mv(part.to, -1)
      if (wrappedLineExtent.begin <= ch && ch < wrappedLineExtent.end) return getRes(ch, moveInStorageOrder)
    }
  }

  // Case 3a: Look for other bidi parts on the same visual line
  let res = searchInVisualLine(partPos + dir, dir, wrappedLineExtent)
  if (res) return res

  // Case 3b: Look for other bidi parts on the next visual line
  let nextCh = dir > 0 ? wrappedLineExtent.end : mv(wrappedLineExtent.begin, -1)
  if (nextCh != null && !(dir > 0 && nextCh == line.text.length)) {
    res = searchInVisualLine(dir > 0 ? 0 : bidi.length - 1, dir, getWrappedLineExtent(nextCh))
    if (res) return res
  }

  // Case 4: Nowhere to move
  return null
}


/***/ }),

/***/ "./node_modules/codemirror/src/line/highlight.js":
/*!*******************************************************!*\
  !*** ./node_modules/codemirror/src/line/highlight.js ***!
  \*******************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "highlightLine": () => (/* binding */ highlightLine),
/* harmony export */   "getLineStyles": () => (/* binding */ getLineStyles),
/* harmony export */   "getContextBefore": () => (/* binding */ getContextBefore),
/* harmony export */   "processLine": () => (/* binding */ processLine),
/* harmony export */   "takeToken": () => (/* binding */ takeToken),
/* harmony export */   "retreatFrontier": () => (/* binding */ retreatFrontier)
/* harmony export */ });
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _modes_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../modes.js */ "./node_modules/codemirror/src/modes.js");
/* harmony import */ var _util_StringStream_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../util/StringStream.js */ "./node_modules/codemirror/src/util/StringStream.js");
/* harmony import */ var _utils_line_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ./utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _pos_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ./pos.js */ "./node_modules/codemirror/src/line/pos.js");







class SavedContext {
  constructor(state, lookAhead) {
    this.state = state
    this.lookAhead = lookAhead
  }
}

class Context {
  constructor(doc, state, line, lookAhead) {
    this.state = state
    this.doc = doc
    this.line = line
    this.maxLookAhead = lookAhead || 0
    this.baseTokens = null
    this.baseTokenPos = 1
  }

  lookAhead(n) {
    let line = this.doc.getLine(this.line + n)
    if (line != null && n > this.maxLookAhead) this.maxLookAhead = n
    return line
  }

  baseToken(n) {
    if (!this.baseTokens) return null
    while (this.baseTokens[this.baseTokenPos] <= n)
      this.baseTokenPos += 2
    let type = this.baseTokens[this.baseTokenPos + 1]
    return {type: type && type.replace(/( |^)overlay .*/, ""),
            size: this.baseTokens[this.baseTokenPos] - n}
  }

  nextLine() {
    this.line++
    if (this.maxLookAhead > 0) this.maxLookAhead--
  }

  static fromSaved(doc, saved, line) {
    if (saved instanceof SavedContext)
      return new Context(doc, (0,_modes_js__WEBPACK_IMPORTED_MODULE_1__.copyState)(doc.mode, saved.state), line, saved.lookAhead)
    else
      return new Context(doc, (0,_modes_js__WEBPACK_IMPORTED_MODULE_1__.copyState)(doc.mode, saved), line)
  }

  save(copy) {
    let state = copy !== false ? (0,_modes_js__WEBPACK_IMPORTED_MODULE_1__.copyState)(this.doc.mode, this.state) : this.state
    return this.maxLookAhead > 0 ? new SavedContext(state, this.maxLookAhead) : state
  }
}


// Compute a style array (an array starting with a mode generation
// -- for invalidation -- followed by pairs of end positions and
// style strings), which is used to highlight the tokens on the
// line.
function highlightLine(cm, line, context, forceToEnd) {
  // A styles array always starts with a number identifying the
  // mode/overlays that it is based on (for easy invalidation).
  let st = [cm.state.modeGen], lineClasses = {}
  // Compute the base array of styles
  runMode(cm, line.text, cm.doc.mode, context, (end, style) => st.push(end, style),
          lineClasses, forceToEnd)
  let state = context.state

  // Run overlays, adjust style array.
  for (let o = 0; o < cm.state.overlays.length; ++o) {
    context.baseTokens = st
    let overlay = cm.state.overlays[o], i = 1, at = 0
    context.state = true
    runMode(cm, line.text, overlay.mode, context, (end, style) => {
      let start = i
      // Ensure there's a token end at the current position, and that i points at it
      while (at < end) {
        let i_end = st[i]
        if (i_end > end)
          st.splice(i, 1, end, st[i+1], i_end)
        i += 2
        at = Math.min(end, i_end)
      }
      if (!style) return
      if (overlay.opaque) {
        st.splice(start, i - start, end, "overlay " + style)
        i = start + 2
      } else {
        for (; start < i; start += 2) {
          let cur = st[start+1]
          st[start+1] = (cur ? cur + " " : "") + "overlay " + style
        }
      }
    }, lineClasses)
    context.state = state
    context.baseTokens = null
    context.baseTokenPos = 1
  }

  return {styles: st, classes: lineClasses.bgClass || lineClasses.textClass ? lineClasses : null}
}

function getLineStyles(cm, line, updateFrontier) {
  if (!line.styles || line.styles[0] != cm.state.modeGen) {
    let context = getContextBefore(cm, (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.lineNo)(line))
    let resetState = line.text.length > cm.options.maxHighlightLength && (0,_modes_js__WEBPACK_IMPORTED_MODULE_1__.copyState)(cm.doc.mode, context.state)
    let result = highlightLine(cm, line, context)
    if (resetState) context.state = resetState
    line.stateAfter = context.save(!resetState)
    line.styles = result.styles
    if (result.classes) line.styleClasses = result.classes
    else if (line.styleClasses) line.styleClasses = null
    if (updateFrontier === cm.doc.highlightFrontier)
      cm.doc.modeFrontier = Math.max(cm.doc.modeFrontier, ++cm.doc.highlightFrontier)
  }
  return line.styles
}

function getContextBefore(cm, n, precise) {
  let doc = cm.doc, display = cm.display
  if (!doc.mode.startState) return new Context(doc, true, n)
  let start = findStartLine(cm, n, precise)
  let saved = start > doc.first && (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, start - 1).stateAfter
  let context = saved ? Context.fromSaved(doc, saved, start) : new Context(doc, (0,_modes_js__WEBPACK_IMPORTED_MODULE_1__.startState)(doc.mode), start)

  doc.iter(start, n, line => {
    processLine(cm, line.text, context)
    let pos = context.line
    line.stateAfter = pos == n - 1 || pos % 5 == 0 || pos >= display.viewFrom && pos < display.viewTo ? context.save() : null
    context.nextLine()
  })
  if (precise) doc.modeFrontier = context.line
  return context
}

// Lightweight form of highlight -- proceed over this line and
// update state, but don't save a style array. Used for lines that
// aren't currently visible.
function processLine(cm, text, context, startAt) {
  let mode = cm.doc.mode
  let stream = new _util_StringStream_js__WEBPACK_IMPORTED_MODULE_2__.default(text, cm.options.tabSize, context)
  stream.start = stream.pos = startAt || 0
  if (text == "") callBlankLine(mode, context.state)
  while (!stream.eol()) {
    readToken(mode, stream, context.state)
    stream.start = stream.pos
  }
}

function callBlankLine(mode, state) {
  if (mode.blankLine) return mode.blankLine(state)
  if (!mode.innerMode) return
  let inner = (0,_modes_js__WEBPACK_IMPORTED_MODULE_1__.innerMode)(mode, state)
  if (inner.mode.blankLine) return inner.mode.blankLine(inner.state)
}

function readToken(mode, stream, state, inner) {
  for (let i = 0; i < 10; i++) {
    if (inner) inner[0] = (0,_modes_js__WEBPACK_IMPORTED_MODULE_1__.innerMode)(mode, state).mode
    let style = mode.token(stream, state)
    if (stream.pos > stream.start) return style
  }
  throw new Error("Mode " + mode.name + " failed to advance stream.")
}

class Token {
  constructor(stream, type, state) {
    this.start = stream.start; this.end = stream.pos
    this.string = stream.current()
    this.type = type || null
    this.state = state
  }
}

// Utility for getTokenAt and getLineTokens
function takeToken(cm, pos, precise, asArray) {
  let doc = cm.doc, mode = doc.mode, style
  pos = (0,_pos_js__WEBPACK_IMPORTED_MODULE_4__.clipPos)(doc, pos)
  let line = (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, pos.line), context = getContextBefore(cm, pos.line, precise)
  let stream = new _util_StringStream_js__WEBPACK_IMPORTED_MODULE_2__.default(line.text, cm.options.tabSize, context), tokens
  if (asArray) tokens = []
  while ((asArray || stream.pos < pos.ch) && !stream.eol()) {
    stream.start = stream.pos
    style = readToken(mode, stream, context.state)
    if (asArray) tokens.push(new Token(stream, style, (0,_modes_js__WEBPACK_IMPORTED_MODULE_1__.copyState)(doc.mode, context.state)))
  }
  return asArray ? tokens : new Token(stream, style, context.state)
}

function extractLineClasses(type, output) {
  if (type) for (;;) {
    let lineClass = type.match(/(?:^|\s+)line-(background-)?(\S+)/)
    if (!lineClass) break
    type = type.slice(0, lineClass.index) + type.slice(lineClass.index + lineClass[0].length)
    let prop = lineClass[1] ? "bgClass" : "textClass"
    if (output[prop] == null)
      output[prop] = lineClass[2]
    else if (!(new RegExp("(?:^|\\s)" + lineClass[2] + "(?:$|\\s)")).test(output[prop]))
      output[prop] += " " + lineClass[2]
  }
  return type
}

// Run the given mode's parser over a line, calling f for each token.
function runMode(cm, text, mode, context, f, lineClasses, forceToEnd) {
  let flattenSpans = mode.flattenSpans
  if (flattenSpans == null) flattenSpans = cm.options.flattenSpans
  let curStart = 0, curStyle = null
  let stream = new _util_StringStream_js__WEBPACK_IMPORTED_MODULE_2__.default(text, cm.options.tabSize, context), style
  let inner = cm.options.addModeClass && [null]
  if (text == "") extractLineClasses(callBlankLine(mode, context.state), lineClasses)
  while (!stream.eol()) {
    if (stream.pos > cm.options.maxHighlightLength) {
      flattenSpans = false
      if (forceToEnd) processLine(cm, text, context, stream.pos)
      stream.pos = text.length
      style = null
    } else {
      style = extractLineClasses(readToken(mode, stream, context.state, inner), lineClasses)
    }
    if (inner) {
      let mName = inner[0].name
      if (mName) style = "m-" + (style ? mName + " " + style : mName)
    }
    if (!flattenSpans || curStyle != style) {
      while (curStart < stream.start) {
        curStart = Math.min(stream.start, curStart + 5000)
        f(curStart, curStyle)
      }
      curStyle = style
    }
    stream.start = stream.pos
  }
  while (curStart < stream.pos) {
    // Webkit seems to refuse to render text nodes longer than 57444
    // characters, and returns inaccurate measurements in nodes
    // starting around 5000 chars.
    let pos = Math.min(stream.pos, curStart + 5000)
    f(pos, curStyle)
    curStart = pos
  }
}

// Finds the line to start with when starting a parse. Tries to
// find a line with a stateAfter, so that it can start with a
// valid state. If that fails, it returns the line with the
// smallest indentation, which tends to need the least context to
// parse correctly.
function findStartLine(cm, n, precise) {
  let minindent, minline, doc = cm.doc
  let lim = precise ? -1 : n - (cm.doc.mode.innerMode ? 1000 : 100)
  for (let search = n; search > lim; --search) {
    if (search <= doc.first) return doc.first
    let line = (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, search - 1), after = line.stateAfter
    if (after && (!precise || search + (after instanceof SavedContext ? after.lookAhead : 0) <= doc.modeFrontier))
      return search
    let indented = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_0__.countColumn)(line.text, null, cm.options.tabSize)
    if (minline == null || minindent > indented) {
      minline = search - 1
      minindent = indented
    }
  }
  return minline
}

function retreatFrontier(doc, n) {
  doc.modeFrontier = Math.min(doc.modeFrontier, n)
  if (doc.highlightFrontier < n - 10) return
  let start = doc.first
  for (let line = n - 1; line > start; line--) {
    let saved = (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, line).stateAfter
    // change is on 3
    // state on line 1 looked ahead 2 -- so saw 3
    // test 1 + 2 < 3 should cover this
    if (saved && (!(saved instanceof SavedContext) || line + saved.lookAhead < n)) {
      start = line + 1
      break
    }
  }
  doc.highlightFrontier = Math.min(doc.highlightFrontier, start)
}


/***/ }),

/***/ "./node_modules/codemirror/src/line/line_data.js":
/*!*******************************************************!*\
  !*** ./node_modules/codemirror/src/line/line_data.js ***!
  \*******************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "Line": () => (/* binding */ Line),
/* harmony export */   "updateLine": () => (/* binding */ updateLine),
/* harmony export */   "cleanUpLine": () => (/* binding */ cleanUpLine),
/* harmony export */   "buildLineContent": () => (/* binding */ buildLineContent),
/* harmony export */   "defaultSpecialCharPlaceholder": () => (/* binding */ defaultSpecialCharPlaceholder),
/* harmony export */   "LineView": () => (/* binding */ LineView),
/* harmony export */   "buildViewArray": () => (/* binding */ buildViewArray)
/* harmony export */ });
/* harmony import */ var _util_bidi_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../util/bidi.js */ "./node_modules/codemirror/src/util/bidi.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _util_feature_detection_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/feature_detection.js */ "./node_modules/codemirror/src/util/feature_detection.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _highlight_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ./highlight.js */ "./node_modules/codemirror/src/line/highlight.js");
/* harmony import */ var _spans_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ./spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _utils_line_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ./utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");











// LINE DATA STRUCTURE

// Line objects. These hold state related to a line, including
// highlighting info (the styles array).
class Line {
  constructor(text, markedSpans, estimateHeight) {
    this.text = text
    ;(0,_spans_js__WEBPACK_IMPORTED_MODULE_7__.attachMarkedSpans)(this, markedSpans)
    this.height = estimateHeight ? estimateHeight(this) : 1
  }

  lineNo() { return (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_8__.lineNo)(this) }
}
(0,_util_event_js__WEBPACK_IMPORTED_MODULE_3__.eventMixin)(Line)

// Change the content (text, markers) of a line. Automatically
// invalidates cached information and tries to re-estimate the
// line's height.
function updateLine(line, text, markedSpans, estimateHeight) {
  line.text = text
  if (line.stateAfter) line.stateAfter = null
  if (line.styles) line.styles = null
  if (line.order != null) line.order = null
  ;(0,_spans_js__WEBPACK_IMPORTED_MODULE_7__.detachMarkedSpans)(line)
  ;(0,_spans_js__WEBPACK_IMPORTED_MODULE_7__.attachMarkedSpans)(line, markedSpans)
  let estHeight = estimateHeight ? estimateHeight(line) : 1
  if (estHeight != line.height) (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_8__.updateLineHeight)(line, estHeight)
}

// Detach a line from the document tree and its markers.
function cleanUpLine(line) {
  line.parent = null
  ;(0,_spans_js__WEBPACK_IMPORTED_MODULE_7__.detachMarkedSpans)(line)
}

// Convert a style as returned by a mode (either null, or a string
// containing one or more styles) to a CSS style. This is cached,
// and also looks for line-wide styles.
let styleToClassCache = {}, styleToClassCacheWithMode = {}
function interpretTokenStyle(style, options) {
  if (!style || /^\s*$/.test(style)) return null
  let cache = options.addModeClass ? styleToClassCacheWithMode : styleToClassCache
  return cache[style] ||
    (cache[style] = style.replace(/\S+/g, "cm-$&"))
}

// Render the DOM representation of the text of a line. Also builds
// up a 'line map', which points at the DOM nodes that represent
// specific stretches of text, and is used by the measuring code.
// The returned object contains the DOM node, this map, and
// information about line-wide styles that were set by the mode.
function buildLineContent(cm, lineView) {
  // The padding-right forces the element to have a 'border', which
  // is needed on Webkit to be able to get line-level bounding
  // rectangles for it (in measureChar).
  let content = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.eltP)("span", null, null, _util_browser_js__WEBPACK_IMPORTED_MODULE_1__.webkit ? "padding-right: .1px" : null)
  let builder = {pre: (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.eltP)("pre", [content], "CodeMirror-line"), content: content,
                 col: 0, pos: 0, cm: cm,
                 trailingSpace: false,
                 splitSpaces: cm.getOption("lineWrapping")}
  lineView.measure = {}

  // Iterate over the logical lines that make up this visual line.
  for (let i = 0; i <= (lineView.rest ? lineView.rest.length : 0); i++) {
    let line = i ? lineView.rest[i - 1] : lineView.line, order
    builder.pos = 0
    builder.addToken = buildToken
    // Optionally wire in some hacks into the token-rendering
    // algorithm, to deal with browser quirks.
    if ((0,_util_feature_detection_js__WEBPACK_IMPORTED_MODULE_4__.hasBadBidiRects)(cm.display.measure) && (order = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_0__.getOrder)(line, cm.doc.direction)))
      builder.addToken = buildTokenBadBidi(builder.addToken, order)
    builder.map = []
    let allowFrontierUpdate = lineView != cm.display.externalMeasured && (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_8__.lineNo)(line)
    insertLineContent(line, builder, (0,_highlight_js__WEBPACK_IMPORTED_MODULE_6__.getLineStyles)(cm, line, allowFrontierUpdate))
    if (line.styleClasses) {
      if (line.styleClasses.bgClass)
        builder.bgClass = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.joinClasses)(line.styleClasses.bgClass, builder.bgClass || "")
      if (line.styleClasses.textClass)
        builder.textClass = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.joinClasses)(line.styleClasses.textClass, builder.textClass || "")
    }

    // Ensure at least a single node is present, for measuring.
    if (builder.map.length == 0)
      builder.map.push(0, 0, builder.content.appendChild((0,_util_feature_detection_js__WEBPACK_IMPORTED_MODULE_4__.zeroWidthElement)(cm.display.measure)))

    // Store the map and a cache object for the current logical line
    if (i == 0) {
      lineView.measure.map = builder.map
      lineView.measure.cache = {}
    } else {
      ;(lineView.measure.maps || (lineView.measure.maps = [])).push(builder.map)
      ;(lineView.measure.caches || (lineView.measure.caches = [])).push({})
    }
  }

  // See issue #2901
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_1__.webkit) {
    let last = builder.content.lastChild
    if (/\bcm-tab\b/.test(last.className) || (last.querySelector && last.querySelector(".cm-tab")))
      builder.content.className = "cm-tab-wrap-hack"
  }

  (0,_util_event_js__WEBPACK_IMPORTED_MODULE_3__.signal)(cm, "renderLine", cm, lineView.line, builder.pre)
  if (builder.pre.className)
    builder.textClass = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.joinClasses)(builder.pre.className, builder.textClass || "")

  return builder
}

function defaultSpecialCharPlaceholder(ch) {
  let token = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.elt)("span", "\u2022", "cm-invalidchar")
  token.title = "\\u" + ch.charCodeAt(0).toString(16)
  token.setAttribute("aria-label", token.title)
  return token
}

// Build up the DOM representation for a single token, and add it to
// the line map. Takes care to render special characters separately.
function buildToken(builder, text, style, startStyle, endStyle, css, attributes) {
  if (!text) return
  let displayText = builder.splitSpaces ? splitSpaces(text, builder.trailingSpace) : text
  let special = builder.cm.state.specialChars, mustWrap = false
  let content
  if (!special.test(text)) {
    builder.col += text.length
    content = document.createTextNode(displayText)
    builder.map.push(builder.pos, builder.pos + text.length, content)
    if (_util_browser_js__WEBPACK_IMPORTED_MODULE_1__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_1__.ie_version < 9) mustWrap = true
    builder.pos += text.length
  } else {
    content = document.createDocumentFragment()
    let pos = 0
    while (true) {
      special.lastIndex = pos
      let m = special.exec(text)
      let skipped = m ? m.index - pos : text.length - pos
      if (skipped) {
        let txt = document.createTextNode(displayText.slice(pos, pos + skipped))
        if (_util_browser_js__WEBPACK_IMPORTED_MODULE_1__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_1__.ie_version < 9) content.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.elt)("span", [txt]))
        else content.appendChild(txt)
        builder.map.push(builder.pos, builder.pos + skipped, txt)
        builder.col += skipped
        builder.pos += skipped
      }
      if (!m) break
      pos += skipped + 1
      let txt
      if (m[0] == "\t") {
        let tabSize = builder.cm.options.tabSize, tabWidth = tabSize - builder.col % tabSize
        txt = content.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.elt)("span", (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_5__.spaceStr)(tabWidth), "cm-tab"))
        txt.setAttribute("role", "presentation")
        txt.setAttribute("cm-text", "\t")
        builder.col += tabWidth
      } else if (m[0] == "\r" || m[0] == "\n") {
        txt = content.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.elt)("span", m[0] == "\r" ? "\u240d" : "\u2424", "cm-invalidchar"))
        txt.setAttribute("cm-text", m[0])
        builder.col += 1
      } else {
        txt = builder.cm.options.specialCharPlaceholder(m[0])
        txt.setAttribute("cm-text", m[0])
        if (_util_browser_js__WEBPACK_IMPORTED_MODULE_1__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_1__.ie_version < 9) content.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.elt)("span", [txt]))
        else content.appendChild(txt)
        builder.col += 1
      }
      builder.map.push(builder.pos, builder.pos + 1, txt)
      builder.pos++
    }
  }
  builder.trailingSpace = displayText.charCodeAt(text.length - 1) == 32
  if (style || startStyle || endStyle || mustWrap || css || attributes) {
    let fullStyle = style || ""
    if (startStyle) fullStyle += startStyle
    if (endStyle) fullStyle += endStyle
    let token = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_2__.elt)("span", [content], fullStyle, css)
    if (attributes) {
      for (let attr in attributes) if (attributes.hasOwnProperty(attr) && attr != "style" && attr != "class")
        token.setAttribute(attr, attributes[attr])
    }
    return builder.content.appendChild(token)
  }
  builder.content.appendChild(content)
}

// Change some spaces to NBSP to prevent the browser from collapsing
// trailing spaces at the end of a line when rendering text (issue #1362).
function splitSpaces(text, trailingBefore) {
  if (text.length > 1 && !/  /.test(text)) return text
  let spaceBefore = trailingBefore, result = ""
  for (let i = 0; i < text.length; i++) {
    let ch = text.charAt(i)
    if (ch == " " && spaceBefore && (i == text.length - 1 || text.charCodeAt(i + 1) == 32))
      ch = "\u00a0"
    result += ch
    spaceBefore = ch == " "
  }
  return result
}

// Work around nonsense dimensions being reported for stretches of
// right-to-left text.
function buildTokenBadBidi(inner, order) {
  return (builder, text, style, startStyle, endStyle, css, attributes) => {
    style = style ? style + " cm-force-border" : "cm-force-border"
    let start = builder.pos, end = start + text.length
    for (;;) {
      // Find the part that overlaps with the start of this text
      let part
      for (let i = 0; i < order.length; i++) {
        part = order[i]
        if (part.to > start && part.from <= start) break
      }
      if (part.to >= end) return inner(builder, text, style, startStyle, endStyle, css, attributes)
      inner(builder, text.slice(0, part.to - start), style, startStyle, null, css, attributes)
      startStyle = null
      text = text.slice(part.to - start)
      start = part.to
    }
  }
}

function buildCollapsedSpan(builder, size, marker, ignoreWidget) {
  let widget = !ignoreWidget && marker.widgetNode
  if (widget) builder.map.push(builder.pos, builder.pos + size, widget)
  if (!ignoreWidget && builder.cm.display.input.needsContentAttribute) {
    if (!widget)
      widget = builder.content.appendChild(document.createElement("span"))
    widget.setAttribute("cm-marker", marker.id)
  }
  if (widget) {
    builder.cm.display.input.setUneditable(widget)
    builder.content.appendChild(widget)
  }
  builder.pos += size
  builder.trailingSpace = false
}

// Outputs a number of spans to make up a line, taking highlighting
// and marked text into account.
function insertLineContent(line, builder, styles) {
  let spans = line.markedSpans, allText = line.text, at = 0
  if (!spans) {
    for (let i = 1; i < styles.length; i+=2)
      builder.addToken(builder, allText.slice(at, at = styles[i]), interpretTokenStyle(styles[i+1], builder.cm.options))
    return
  }

  let len = allText.length, pos = 0, i = 1, text = "", style, css
  let nextChange = 0, spanStyle, spanEndStyle, spanStartStyle, collapsed, attributes
  for (;;) {
    if (nextChange == pos) { // Update current marker set
      spanStyle = spanEndStyle = spanStartStyle = css = ""
      attributes = null
      collapsed = null; nextChange = Infinity
      let foundBookmarks = [], endStyles
      for (let j = 0; j < spans.length; ++j) {
        let sp = spans[j], m = sp.marker
        if (m.type == "bookmark" && sp.from == pos && m.widgetNode) {
          foundBookmarks.push(m)
        } else if (sp.from <= pos && (sp.to == null || sp.to > pos || m.collapsed && sp.to == pos && sp.from == pos)) {
          if (sp.to != null && sp.to != pos && nextChange > sp.to) {
            nextChange = sp.to
            spanEndStyle = ""
          }
          if (m.className) spanStyle += " " + m.className
          if (m.css) css = (css ? css + ";" : "") + m.css
          if (m.startStyle && sp.from == pos) spanStartStyle += " " + m.startStyle
          if (m.endStyle && sp.to == nextChange) (endStyles || (endStyles = [])).push(m.endStyle, sp.to)
          // support for the old title property
          // https://github.com/codemirror/CodeMirror/pull/5673
          if (m.title) (attributes || (attributes = {})).title = m.title
          if (m.attributes) {
            for (let attr in m.attributes)
              (attributes || (attributes = {}))[attr] = m.attributes[attr]
          }
          if (m.collapsed && (!collapsed || (0,_spans_js__WEBPACK_IMPORTED_MODULE_7__.compareCollapsedMarkers)(collapsed.marker, m) < 0))
            collapsed = sp
        } else if (sp.from > pos && nextChange > sp.from) {
          nextChange = sp.from
        }
      }
      if (endStyles) for (let j = 0; j < endStyles.length; j += 2)
        if (endStyles[j + 1] == nextChange) spanEndStyle += " " + endStyles[j]

      if (!collapsed || collapsed.from == pos) for (let j = 0; j < foundBookmarks.length; ++j)
        buildCollapsedSpan(builder, 0, foundBookmarks[j])
      if (collapsed && (collapsed.from || 0) == pos) {
        buildCollapsedSpan(builder, (collapsed.to == null ? len + 1 : collapsed.to) - pos,
                           collapsed.marker, collapsed.from == null)
        if (collapsed.to == null) return
        if (collapsed.to == pos) collapsed = false
      }
    }
    if (pos >= len) break

    let upto = Math.min(len, nextChange)
    while (true) {
      if (text) {
        let end = pos + text.length
        if (!collapsed) {
          let tokenText = end > upto ? text.slice(0, upto - pos) : text
          builder.addToken(builder, tokenText, style ? style + spanStyle : spanStyle,
                           spanStartStyle, pos + tokenText.length == nextChange ? spanEndStyle : "", css, attributes)
        }
        if (end >= upto) {text = text.slice(upto - pos); pos = upto; break}
        pos = end
        spanStartStyle = ""
      }
      text = allText.slice(at, at = styles[i++])
      style = interpretTokenStyle(styles[i++], builder.cm.options)
    }
  }
}


// These objects are used to represent the visible (currently drawn)
// part of the document. A LineView may correspond to multiple
// logical lines, if those are connected by collapsed ranges.
function LineView(doc, line, lineN) {
  // The starting line
  this.line = line
  // Continuing lines, if any
  this.rest = (0,_spans_js__WEBPACK_IMPORTED_MODULE_7__.visualLineContinued)(line)
  // Number of logical lines in this visual line
  this.size = this.rest ? (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_8__.lineNo)((0,_util_misc_js__WEBPACK_IMPORTED_MODULE_5__.lst)(this.rest)) - lineN + 1 : 1
  this.node = this.text = null
  this.hidden = (0,_spans_js__WEBPACK_IMPORTED_MODULE_7__.lineIsHidden)(doc, line)
}

// Create a range of LineView objects for the given lines.
function buildViewArray(cm, from, to) {
  let array = [], nextPos
  for (let pos = from; pos < to; pos = nextPos) {
    let view = new LineView(cm.doc, (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_8__.getLine)(cm.doc, pos), pos)
    nextPos = pos + view.size
    array.push(view)
  }
  return array
}


/***/ }),

/***/ "./node_modules/codemirror/src/line/pos.js":
/*!*************************************************!*\
  !*** ./node_modules/codemirror/src/line/pos.js ***!
  \*************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "Pos": () => (/* binding */ Pos),
/* harmony export */   "cmp": () => (/* binding */ cmp),
/* harmony export */   "equalCursorPos": () => (/* binding */ equalCursorPos),
/* harmony export */   "copyPos": () => (/* binding */ copyPos),
/* harmony export */   "maxPos": () => (/* binding */ maxPos),
/* harmony export */   "minPos": () => (/* binding */ minPos),
/* harmony export */   "clipLine": () => (/* binding */ clipLine),
/* harmony export */   "clipPos": () => (/* binding */ clipPos),
/* harmony export */   "clipPosArray": () => (/* binding */ clipPosArray)
/* harmony export */ });
/* harmony import */ var _utils_line_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");


// A Pos instance represents a position within the text.
function Pos(line, ch, sticky = null) {
  if (!(this instanceof Pos)) return new Pos(line, ch, sticky)
  this.line = line
  this.ch = ch
  this.sticky = sticky
}

// Compare two positions, return 0 if they are the same, a negative
// number when a is less, and a positive number otherwise.
function cmp(a, b) { return a.line - b.line || a.ch - b.ch }

function equalCursorPos(a, b) { return a.sticky == b.sticky && cmp(a, b) == 0 }

function copyPos(x) {return Pos(x.line, x.ch)}
function maxPos(a, b) { return cmp(a, b) < 0 ? b : a }
function minPos(a, b) { return cmp(a, b) < 0 ? a : b }

// Most of the external API clips given positions to make sure they
// actually exist within the document.
function clipLine(doc, n) {return Math.max(doc.first, Math.min(n, doc.first + doc.size - 1))}
function clipPos(doc, pos) {
  if (pos.line < doc.first) return Pos(doc.first, 0)
  let last = doc.first + doc.size - 1
  if (pos.line > last) return Pos(last, (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_0__.getLine)(doc, last).text.length)
  return clipToLen(pos, (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_0__.getLine)(doc, pos.line).text.length)
}
function clipToLen(pos, linelen) {
  let ch = pos.ch
  if (ch == null || ch > linelen) return Pos(pos.line, linelen)
  else if (ch < 0) return Pos(pos.line, 0)
  else return pos
}
function clipPosArray(doc, array) {
  let out = []
  for (let i = 0; i < array.length; i++) out[i] = clipPos(doc, array[i])
  return out
}


/***/ }),

/***/ "./node_modules/codemirror/src/line/saw_special_spans.js":
/*!***************************************************************!*\
  !*** ./node_modules/codemirror/src/line/saw_special_spans.js ***!
  \***************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "sawReadOnlySpans": () => (/* binding */ sawReadOnlySpans),
/* harmony export */   "sawCollapsedSpans": () => (/* binding */ sawCollapsedSpans),
/* harmony export */   "seeReadOnlySpans": () => (/* binding */ seeReadOnlySpans),
/* harmony export */   "seeCollapsedSpans": () => (/* binding */ seeCollapsedSpans)
/* harmony export */ });
// Optimize some code when these features are not used.
let sawReadOnlySpans = false, sawCollapsedSpans = false

function seeReadOnlySpans() {
  sawReadOnlySpans = true
}

function seeCollapsedSpans() {
  sawCollapsedSpans = true
}


/***/ }),

/***/ "./node_modules/codemirror/src/line/spans.js":
/*!***************************************************!*\
  !*** ./node_modules/codemirror/src/line/spans.js ***!
  \***************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "MarkedSpan": () => (/* binding */ MarkedSpan),
/* harmony export */   "getMarkedSpanFor": () => (/* binding */ getMarkedSpanFor),
/* harmony export */   "removeMarkedSpan": () => (/* binding */ removeMarkedSpan),
/* harmony export */   "addMarkedSpan": () => (/* binding */ addMarkedSpan),
/* harmony export */   "stretchSpansOverChange": () => (/* binding */ stretchSpansOverChange),
/* harmony export */   "removeReadOnlyRanges": () => (/* binding */ removeReadOnlyRanges),
/* harmony export */   "detachMarkedSpans": () => (/* binding */ detachMarkedSpans),
/* harmony export */   "attachMarkedSpans": () => (/* binding */ attachMarkedSpans),
/* harmony export */   "compareCollapsedMarkers": () => (/* binding */ compareCollapsedMarkers),
/* harmony export */   "collapsedSpanAtStart": () => (/* binding */ collapsedSpanAtStart),
/* harmony export */   "collapsedSpanAtEnd": () => (/* binding */ collapsedSpanAtEnd),
/* harmony export */   "collapsedSpanAround": () => (/* binding */ collapsedSpanAround),
/* harmony export */   "conflictingCollapsedRange": () => (/* binding */ conflictingCollapsedRange),
/* harmony export */   "visualLine": () => (/* binding */ visualLine),
/* harmony export */   "visualLineEnd": () => (/* binding */ visualLineEnd),
/* harmony export */   "visualLineContinued": () => (/* binding */ visualLineContinued),
/* harmony export */   "visualLineNo": () => (/* binding */ visualLineNo),
/* harmony export */   "visualLineEndNo": () => (/* binding */ visualLineEndNo),
/* harmony export */   "lineIsHidden": () => (/* binding */ lineIsHidden),
/* harmony export */   "heightAtLine": () => (/* binding */ heightAtLine),
/* harmony export */   "lineLength": () => (/* binding */ lineLength),
/* harmony export */   "findMaxLine": () => (/* binding */ findMaxLine)
/* harmony export */ });
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _pos_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ./pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _saw_special_spans_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ./saw_special_spans.js */ "./node_modules/codemirror/src/line/saw_special_spans.js");
/* harmony import */ var _utils_line_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ./utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");






// TEXTMARKER SPANS

function MarkedSpan(marker, from, to) {
  this.marker = marker
  this.from = from; this.to = to
}

// Search an array of spans for a span matching the given marker.
function getMarkedSpanFor(spans, marker) {
  if (spans) for (let i = 0; i < spans.length; ++i) {
    let span = spans[i]
    if (span.marker == marker) return span
  }
}

// Remove a span from an array, returning undefined if no spans are
// left (we don't store arrays for lines without spans).
function removeMarkedSpan(spans, span) {
  let r
  for (let i = 0; i < spans.length; ++i)
    if (spans[i] != span) (r || (r = [])).push(spans[i])
  return r
}

// Add a span to a line.
function addMarkedSpan(line, span, op) {
  let inThisOp = op && window.WeakSet && (op.markedSpans || (op.markedSpans = new WeakSet))
  if (inThisOp && inThisOp.has(line.markedSpans)) {
    line.markedSpans.push(span)
  } else {
    line.markedSpans = line.markedSpans ? line.markedSpans.concat([span]) : [span]
    if (inThisOp) inThisOp.add(line.markedSpans)
  }
  span.marker.attachLine(line)
}

// Used for the algorithm that adjusts markers for a change in the
// document. These functions cut an array of spans at a given
// character position, returning an array of remaining chunks (or
// undefined if nothing remains).
function markedSpansBefore(old, startCh, isInsert) {
  let nw
  if (old) for (let i = 0; i < old.length; ++i) {
    let span = old[i], marker = span.marker
    let startsBefore = span.from == null || (marker.inclusiveLeft ? span.from <= startCh : span.from < startCh)
    if (startsBefore || span.from == startCh && marker.type == "bookmark" && (!isInsert || !span.marker.insertLeft)) {
      let endsAfter = span.to == null || (marker.inclusiveRight ? span.to >= startCh : span.to > startCh)
      ;(nw || (nw = [])).push(new MarkedSpan(marker, span.from, endsAfter ? null : span.to))
    }
  }
  return nw
}
function markedSpansAfter(old, endCh, isInsert) {
  let nw
  if (old) for (let i = 0; i < old.length; ++i) {
    let span = old[i], marker = span.marker
    let endsAfter = span.to == null || (marker.inclusiveRight ? span.to >= endCh : span.to > endCh)
    if (endsAfter || span.from == endCh && marker.type == "bookmark" && (!isInsert || span.marker.insertLeft)) {
      let startsBefore = span.from == null || (marker.inclusiveLeft ? span.from <= endCh : span.from < endCh)
      ;(nw || (nw = [])).push(new MarkedSpan(marker, startsBefore ? null : span.from - endCh,
                                            span.to == null ? null : span.to - endCh))
    }
  }
  return nw
}

// Given a change object, compute the new set of marker spans that
// cover the line in which the change took place. Removes spans
// entirely within the change, reconnects spans belonging to the
// same marker that appear on both sides of the change, and cuts off
// spans partially within the change. Returns an array of span
// arrays with one element for each line in (after) the change.
function stretchSpansOverChange(doc, change) {
  if (change.full) return null
  let oldFirst = (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.isLine)(doc, change.from.line) && (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, change.from.line).markedSpans
  let oldLast = (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.isLine)(doc, change.to.line) && (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, change.to.line).markedSpans
  if (!oldFirst && !oldLast) return null

  let startCh = change.from.ch, endCh = change.to.ch, isInsert = (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(change.from, change.to) == 0
  // Get the spans that 'stick out' on both sides
  let first = markedSpansBefore(oldFirst, startCh, isInsert)
  let last = markedSpansAfter(oldLast, endCh, isInsert)

  // Next, merge those two ends
  let sameLine = change.text.length == 1, offset = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_0__.lst)(change.text).length + (sameLine ? startCh : 0)
  if (first) {
    // Fix up .to properties of first
    for (let i = 0; i < first.length; ++i) {
      let span = first[i]
      if (span.to == null) {
        let found = getMarkedSpanFor(last, span.marker)
        if (!found) span.to = startCh
        else if (sameLine) span.to = found.to == null ? null : found.to + offset
      }
    }
  }
  if (last) {
    // Fix up .from in last (or move them into first in case of sameLine)
    for (let i = 0; i < last.length; ++i) {
      let span = last[i]
      if (span.to != null) span.to += offset
      if (span.from == null) {
        let found = getMarkedSpanFor(first, span.marker)
        if (!found) {
          span.from = offset
          if (sameLine) (first || (first = [])).push(span)
        }
      } else {
        span.from += offset
        if (sameLine) (first || (first = [])).push(span)
      }
    }
  }
  // Make sure we didn't create any zero-length spans
  if (first) first = clearEmptySpans(first)
  if (last && last != first) last = clearEmptySpans(last)

  let newMarkers = [first]
  if (!sameLine) {
    // Fill gap with whole-line-spans
    let gap = change.text.length - 2, gapMarkers
    if (gap > 0 && first)
      for (let i = 0; i < first.length; ++i)
        if (first[i].to == null)
          (gapMarkers || (gapMarkers = [])).push(new MarkedSpan(first[i].marker, null, null))
    for (let i = 0; i < gap; ++i)
      newMarkers.push(gapMarkers)
    newMarkers.push(last)
  }
  return newMarkers
}

// Remove spans that are empty and don't have a clearWhenEmpty
// option of false.
function clearEmptySpans(spans) {
  for (let i = 0; i < spans.length; ++i) {
    let span = spans[i]
    if (span.from != null && span.from == span.to && span.marker.clearWhenEmpty !== false)
      spans.splice(i--, 1)
  }
  if (!spans.length) return null
  return spans
}

// Used to 'clip' out readOnly ranges when making a change.
function removeReadOnlyRanges(doc, from, to) {
  let markers = null
  doc.iter(from.line, to.line + 1, line => {
    if (line.markedSpans) for (let i = 0; i < line.markedSpans.length; ++i) {
      let mark = line.markedSpans[i].marker
      if (mark.readOnly && (!markers || (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_0__.indexOf)(markers, mark) == -1))
        (markers || (markers = [])).push(mark)
    }
  })
  if (!markers) return null
  let parts = [{from: from, to: to}]
  for (let i = 0; i < markers.length; ++i) {
    let mk = markers[i], m = mk.find(0)
    for (let j = 0; j < parts.length; ++j) {
      let p = parts[j]
      if ((0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(p.to, m.from) < 0 || (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(p.from, m.to) > 0) continue
      let newParts = [j, 1], dfrom = (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(p.from, m.from), dto = (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(p.to, m.to)
      if (dfrom < 0 || !mk.inclusiveLeft && !dfrom)
        newParts.push({from: p.from, to: m.from})
      if (dto > 0 || !mk.inclusiveRight && !dto)
        newParts.push({from: m.to, to: p.to})
      parts.splice.apply(parts, newParts)
      j += newParts.length - 3
    }
  }
  return parts
}

// Connect or disconnect spans from a line.
function detachMarkedSpans(line) {
  let spans = line.markedSpans
  if (!spans) return
  for (let i = 0; i < spans.length; ++i)
    spans[i].marker.detachLine(line)
  line.markedSpans = null
}
function attachMarkedSpans(line, spans) {
  if (!spans) return
  for (let i = 0; i < spans.length; ++i)
    spans[i].marker.attachLine(line)
  line.markedSpans = spans
}

// Helpers used when computing which overlapping collapsed span
// counts as the larger one.
function extraLeft(marker) { return marker.inclusiveLeft ? -1 : 0 }
function extraRight(marker) { return marker.inclusiveRight ? 1 : 0 }

// Returns a number indicating which of two overlapping collapsed
// spans is larger (and thus includes the other). Falls back to
// comparing ids when the spans cover exactly the same range.
function compareCollapsedMarkers(a, b) {
  let lenDiff = a.lines.length - b.lines.length
  if (lenDiff != 0) return lenDiff
  let aPos = a.find(), bPos = b.find()
  let fromCmp = (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(aPos.from, bPos.from) || extraLeft(a) - extraLeft(b)
  if (fromCmp) return -fromCmp
  let toCmp = (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(aPos.to, bPos.to) || extraRight(a) - extraRight(b)
  if (toCmp) return toCmp
  return b.id - a.id
}

// Find out whether a line ends or starts in a collapsed span. If
// so, return the marker for that span.
function collapsedSpanAtSide(line, start) {
  let sps = _saw_special_spans_js__WEBPACK_IMPORTED_MODULE_2__.sawCollapsedSpans && line.markedSpans, found
  if (sps) for (let sp, i = 0; i < sps.length; ++i) {
    sp = sps[i]
    if (sp.marker.collapsed && (start ? sp.from : sp.to) == null &&
        (!found || compareCollapsedMarkers(found, sp.marker) < 0))
      found = sp.marker
  }
  return found
}
function collapsedSpanAtStart(line) { return collapsedSpanAtSide(line, true) }
function collapsedSpanAtEnd(line) { return collapsedSpanAtSide(line, false) }

function collapsedSpanAround(line, ch) {
  let sps = _saw_special_spans_js__WEBPACK_IMPORTED_MODULE_2__.sawCollapsedSpans && line.markedSpans, found
  if (sps) for (let i = 0; i < sps.length; ++i) {
    let sp = sps[i]
    if (sp.marker.collapsed && (sp.from == null || sp.from < ch) && (sp.to == null || sp.to > ch) &&
        (!found || compareCollapsedMarkers(found, sp.marker) < 0)) found = sp.marker
  }
  return found
}

// Test whether there exists a collapsed span that partially
// overlaps (covers the start or end, but not both) of a new span.
// Such overlap is not allowed.
function conflictingCollapsedRange(doc, lineNo, from, to, marker) {
  let line = (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, lineNo)
  let sps = _saw_special_spans_js__WEBPACK_IMPORTED_MODULE_2__.sawCollapsedSpans && line.markedSpans
  if (sps) for (let i = 0; i < sps.length; ++i) {
    let sp = sps[i]
    if (!sp.marker.collapsed) continue
    let found = sp.marker.find(0)
    let fromCmp = (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(found.from, from) || extraLeft(sp.marker) - extraLeft(marker)
    let toCmp = (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(found.to, to) || extraRight(sp.marker) - extraRight(marker)
    if (fromCmp >= 0 && toCmp <= 0 || fromCmp <= 0 && toCmp >= 0) continue
    if (fromCmp <= 0 && (sp.marker.inclusiveRight && marker.inclusiveLeft ? (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(found.to, from) >= 0 : (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(found.to, from) > 0) ||
        fromCmp >= 0 && (sp.marker.inclusiveRight && marker.inclusiveLeft ? (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(found.from, to) <= 0 : (0,_pos_js__WEBPACK_IMPORTED_MODULE_1__.cmp)(found.from, to) < 0))
      return true
  }
}

// A visual line is a line as drawn on the screen. Folding, for
// example, can cause multiple logical lines to appear on the same
// visual line. This finds the start of the visual line that the
// given line is part of (usually that is the line itself).
function visualLine(line) {
  let merged
  while (merged = collapsedSpanAtStart(line))
    line = merged.find(-1, true).line
  return line
}

function visualLineEnd(line) {
  let merged
  while (merged = collapsedSpanAtEnd(line))
    line = merged.find(1, true).line
  return line
}

// Returns an array of logical lines that continue the visual line
// started by the argument, or undefined if there are no such lines.
function visualLineContinued(line) {
  let merged, lines
  while (merged = collapsedSpanAtEnd(line)) {
    line = merged.find(1, true).line
    ;(lines || (lines = [])).push(line)
  }
  return lines
}

// Get the line number of the start of the visual line that the
// given line number is part of.
function visualLineNo(doc, lineN) {
  let line = (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, lineN), vis = visualLine(line)
  if (line == vis) return lineN
  return (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.lineNo)(vis)
}

// Get the line number of the start of the next visual line after
// the given line.
function visualLineEndNo(doc, lineN) {
  if (lineN > doc.lastLine()) return lineN
  let line = (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, lineN), merged
  if (!lineIsHidden(doc, line)) return lineN
  while (merged = collapsedSpanAtEnd(line))
    line = merged.find(1, true).line
  return (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.lineNo)(line) + 1
}

// Compute whether a line is hidden. Lines count as hidden when they
// are part of a visual line that starts with another line, or when
// they are entirely covered by collapsed, non-widget span.
function lineIsHidden(doc, line) {
  let sps = _saw_special_spans_js__WEBPACK_IMPORTED_MODULE_2__.sawCollapsedSpans && line.markedSpans
  if (sps) for (let sp, i = 0; i < sps.length; ++i) {
    sp = sps[i]
    if (!sp.marker.collapsed) continue
    if (sp.from == null) return true
    if (sp.marker.widgetNode) continue
    if (sp.from == 0 && sp.marker.inclusiveLeft && lineIsHiddenInner(doc, line, sp))
      return true
  }
}
function lineIsHiddenInner(doc, line, span) {
  if (span.to == null) {
    let end = span.marker.find(1, true)
    return lineIsHiddenInner(doc, end.line, getMarkedSpanFor(end.line.markedSpans, span.marker))
  }
  if (span.marker.inclusiveRight && span.to == line.text.length)
    return true
  for (let sp, i = 0; i < line.markedSpans.length; ++i) {
    sp = line.markedSpans[i]
    if (sp.marker.collapsed && !sp.marker.widgetNode && sp.from == span.to &&
        (sp.to == null || sp.to != span.from) &&
        (sp.marker.inclusiveLeft || span.marker.inclusiveRight) &&
        lineIsHiddenInner(doc, line, sp)) return true
  }
}

// Find the height above the given line.
function heightAtLine(lineObj) {
  lineObj = visualLine(lineObj)

  let h = 0, chunk = lineObj.parent
  for (let i = 0; i < chunk.lines.length; ++i) {
    let line = chunk.lines[i]
    if (line == lineObj) break
    else h += line.height
  }
  for (let p = chunk.parent; p; chunk = p, p = chunk.parent) {
    for (let i = 0; i < p.children.length; ++i) {
      let cur = p.children[i]
      if (cur == chunk) break
      else h += cur.height
    }
  }
  return h
}

// Compute the character length of a line, taking into account
// collapsed ranges (see markText) that might hide parts, and join
// other lines onto it.
function lineLength(line) {
  if (line.height == 0) return 0
  let len = line.text.length, merged, cur = line
  while (merged = collapsedSpanAtStart(cur)) {
    let found = merged.find(0, true)
    cur = found.from.line
    len += found.from.ch - found.to.ch
  }
  cur = line
  while (merged = collapsedSpanAtEnd(cur)) {
    let found = merged.find(0, true)
    len -= cur.text.length - found.from.ch
    cur = found.to.line
    len += cur.text.length - found.to.ch
  }
  return len
}

// Find the longest line in the document.
function findMaxLine(cm) {
  let d = cm.display, doc = cm.doc
  d.maxLine = (0,_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, doc.first)
  d.maxLineLength = lineLength(d.maxLine)
  d.maxLineChanged = true
  doc.iter(line => {
    let len = lineLength(line)
    if (len > d.maxLineLength) {
      d.maxLineLength = len
      d.maxLine = line
    }
  })
}


/***/ }),

/***/ "./node_modules/codemirror/src/line/utils_line.js":
/*!********************************************************!*\
  !*** ./node_modules/codemirror/src/line/utils_line.js ***!
  \********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "getLine": () => (/* binding */ getLine),
/* harmony export */   "getBetween": () => (/* binding */ getBetween),
/* harmony export */   "getLines": () => (/* binding */ getLines),
/* harmony export */   "updateLineHeight": () => (/* binding */ updateLineHeight),
/* harmony export */   "lineNo": () => (/* binding */ lineNo),
/* harmony export */   "lineAtHeight": () => (/* binding */ lineAtHeight),
/* harmony export */   "isLine": () => (/* binding */ isLine),
/* harmony export */   "lineNumberFor": () => (/* binding */ lineNumberFor)
/* harmony export */ });
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");


// Find the line object corresponding to the given line number.
function getLine(doc, n) {
  n -= doc.first
  if (n < 0 || n >= doc.size) throw new Error("There is no line " + (n + doc.first) + " in the document.")
  let chunk = doc
  while (!chunk.lines) {
    for (let i = 0;; ++i) {
      let child = chunk.children[i], sz = child.chunkSize()
      if (n < sz) { chunk = child; break }
      n -= sz
    }
  }
  return chunk.lines[n]
}

// Get the part of a document between two positions, as an array of
// strings.
function getBetween(doc, start, end) {
  let out = [], n = start.line
  doc.iter(start.line, end.line + 1, line => {
    let text = line.text
    if (n == end.line) text = text.slice(0, end.ch)
    if (n == start.line) text = text.slice(start.ch)
    out.push(text)
    ++n
  })
  return out
}
// Get the lines between from and to, as array of strings.
function getLines(doc, from, to) {
  let out = []
  doc.iter(from, to, line => { out.push(line.text) }) // iter aborts when callback returns truthy value
  return out
}

// Update the height of a line, propagating the height change
// upwards to parent nodes.
function updateLineHeight(line, height) {
  let diff = height - line.height
  if (diff) for (let n = line; n; n = n.parent) n.height += diff
}

// Given a line object, find its line number by walking up through
// its parent links.
function lineNo(line) {
  if (line.parent == null) return null
  let cur = line.parent, no = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_0__.indexOf)(cur.lines, line)
  for (let chunk = cur.parent; chunk; cur = chunk, chunk = chunk.parent) {
    for (let i = 0;; ++i) {
      if (chunk.children[i] == cur) break
      no += chunk.children[i].chunkSize()
    }
  }
  return no + cur.first
}

// Find the line at the given vertical position, using the height
// information in the document tree.
function lineAtHeight(chunk, h) {
  let n = chunk.first
  outer: do {
    for (let i = 0; i < chunk.children.length; ++i) {
      let child = chunk.children[i], ch = child.height
      if (h < ch) { chunk = child; continue outer }
      h -= ch
      n += child.chunkSize()
    }
    return n
  } while (!chunk.lines)
  let i = 0
  for (; i < chunk.lines.length; ++i) {
    let line = chunk.lines[i], lh = line.height
    if (h < lh) break
    h -= lh
  }
  return n + i
}

function isLine(doc, l) {return l >= doc.first && l < doc.first + doc.size}

function lineNumberFor(options, i) {
  return String(options.lineNumberFormatter(i + options.firstLineNumber))
}


/***/ }),

/***/ "./node_modules/codemirror/src/measurement/position_measurement.js":
/*!*************************************************************************!*\
  !*** ./node_modules/codemirror/src/measurement/position_measurement.js ***!
  \*************************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "paddingTop": () => (/* binding */ paddingTop),
/* harmony export */   "paddingVert": () => (/* binding */ paddingVert),
/* harmony export */   "paddingH": () => (/* binding */ paddingH),
/* harmony export */   "scrollGap": () => (/* binding */ scrollGap),
/* harmony export */   "displayWidth": () => (/* binding */ displayWidth),
/* harmony export */   "displayHeight": () => (/* binding */ displayHeight),
/* harmony export */   "mapFromLineView": () => (/* binding */ mapFromLineView),
/* harmony export */   "measureChar": () => (/* binding */ measureChar),
/* harmony export */   "findViewForLine": () => (/* binding */ findViewForLine),
/* harmony export */   "prepareMeasureForLine": () => (/* binding */ prepareMeasureForLine),
/* harmony export */   "measureCharPrepared": () => (/* binding */ measureCharPrepared),
/* harmony export */   "nodeAndOffsetInLineMap": () => (/* binding */ nodeAndOffsetInLineMap),
/* harmony export */   "clearLineMeasurementCacheFor": () => (/* binding */ clearLineMeasurementCacheFor),
/* harmony export */   "clearLineMeasurementCache": () => (/* binding */ clearLineMeasurementCache),
/* harmony export */   "clearCaches": () => (/* binding */ clearCaches),
/* harmony export */   "intoCoordSystem": () => (/* binding */ intoCoordSystem),
/* harmony export */   "fromCoordSystem": () => (/* binding */ fromCoordSystem),
/* harmony export */   "charCoords": () => (/* binding */ charCoords),
/* harmony export */   "cursorCoords": () => (/* binding */ cursorCoords),
/* harmony export */   "estimateCoords": () => (/* binding */ estimateCoords),
/* harmony export */   "coordsChar": () => (/* binding */ coordsChar),
/* harmony export */   "wrappedLineExtentChar": () => (/* binding */ wrappedLineExtentChar),
/* harmony export */   "textHeight": () => (/* binding */ textHeight),
/* harmony export */   "charWidth": () => (/* binding */ charWidth),
/* harmony export */   "getDimensions": () => (/* binding */ getDimensions),
/* harmony export */   "compensateForHScroll": () => (/* binding */ compensateForHScroll),
/* harmony export */   "estimateHeight": () => (/* binding */ estimateHeight),
/* harmony export */   "estimateLineHeights": () => (/* binding */ estimateLineHeights),
/* harmony export */   "posFromMouse": () => (/* binding */ posFromMouse),
/* harmony export */   "findViewIndex": () => (/* binding */ findViewIndex)
/* harmony export */ });
/* harmony import */ var _line_line_data_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/line_data.js */ "./node_modules/codemirror/src/line/line_data.js");
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _util_bidi_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/bidi.js */ "./node_modules/codemirror/src/util/bidi.js");
/* harmony import */ var _util_browser_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../util/browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _util_feature_detection_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ../util/feature_detection.js */ "./node_modules/codemirror/src/util/feature_detection.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _display_update_line_js__WEBPACK_IMPORTED_MODULE_10__ = __webpack_require__(/*! ../display/update_line.js */ "./node_modules/codemirror/src/display/update_line.js");
/* harmony import */ var _widgets_js__WEBPACK_IMPORTED_MODULE_11__ = __webpack_require__(/*! ./widgets.js */ "./node_modules/codemirror/src/measurement/widgets.js");














// POSITION MEASUREMENT

function paddingTop(display) {return display.lineSpace.offsetTop}
function paddingVert(display) {return display.mover.offsetHeight - display.lineSpace.offsetHeight}
function paddingH(display) {
  if (display.cachedPaddingH) return display.cachedPaddingH
  let e = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.removeChildrenAndAdd)(display.measure, (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.elt)("pre", "x", "CodeMirror-line-like"))
  let style = window.getComputedStyle ? window.getComputedStyle(e) : e.currentStyle
  let data = {left: parseInt(style.paddingLeft), right: parseInt(style.paddingRight)}
  if (!isNaN(data.left) && !isNaN(data.right)) display.cachedPaddingH = data
  return data
}

function scrollGap(cm) { return _util_misc_js__WEBPACK_IMPORTED_MODULE_9__.scrollerGap - cm.display.nativeBarWidth }
function displayWidth(cm) {
  return cm.display.scroller.clientWidth - scrollGap(cm) - cm.display.barWidth
}
function displayHeight(cm) {
  return cm.display.scroller.clientHeight - scrollGap(cm) - cm.display.barHeight
}

// Ensure the lineView.wrapping.heights array is populated. This is
// an array of bottom offsets for the lines that make up a drawn
// line. When lineWrapping is on, there might be more than one
// height.
function ensureLineHeights(cm, lineView, rect) {
  let wrapping = cm.options.lineWrapping
  let curWidth = wrapping && displayWidth(cm)
  if (!lineView.measure.heights || wrapping && lineView.measure.width != curWidth) {
    let heights = lineView.measure.heights = []
    if (wrapping) {
      lineView.measure.width = curWidth
      let rects = lineView.text.firstChild.getClientRects()
      for (let i = 0; i < rects.length - 1; i++) {
        let cur = rects[i], next = rects[i + 1]
        if (Math.abs(cur.bottom - next.bottom) > 2)
          heights.push((cur.bottom + next.top) / 2 - rect.top)
      }
    }
    heights.push(rect.bottom - rect.top)
  }
}

// Find a line map (mapping character offsets to text nodes) and a
// measurement cache for the given line number. (A line view might
// contain multiple lines when collapsed ranges are present.)
function mapFromLineView(lineView, line, lineN) {
  if (lineView.line == line)
    return {map: lineView.measure.map, cache: lineView.measure.cache}
  for (let i = 0; i < lineView.rest.length; i++)
    if (lineView.rest[i] == line)
      return {map: lineView.measure.maps[i], cache: lineView.measure.caches[i]}
  for (let i = 0; i < lineView.rest.length; i++)
    if ((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.lineNo)(lineView.rest[i]) > lineN)
      return {map: lineView.measure.maps[i], cache: lineView.measure.caches[i], before: true}
}

// Render a line into the hidden node display.externalMeasured. Used
// when measurement is needed for a line that's not in the viewport.
function updateExternalMeasurement(cm, line) {
  line = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_2__.visualLine)(line)
  let lineN = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.lineNo)(line)
  let view = cm.display.externalMeasured = new _line_line_data_js__WEBPACK_IMPORTED_MODULE_0__.LineView(cm.doc, line, lineN)
  view.lineN = lineN
  let built = view.built = (0,_line_line_data_js__WEBPACK_IMPORTED_MODULE_0__.buildLineContent)(cm, view)
  view.text = built.pre
  ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.removeChildrenAndAdd)(cm.display.lineMeasure, built.pre)
  return view
}

// Get a {top, bottom, left, right} box (in line-local coordinates)
// for a given character.
function measureChar(cm, line, ch, bias) {
  return measureCharPrepared(cm, prepareMeasureForLine(cm, line), ch, bias)
}

// Find a line view that corresponds to the given line number.
function findViewForLine(cm, lineN) {
  if (lineN >= cm.display.viewFrom && lineN < cm.display.viewTo)
    return cm.display.view[findViewIndex(cm, lineN)]
  let ext = cm.display.externalMeasured
  if (ext && lineN >= ext.lineN && lineN < ext.lineN + ext.size)
    return ext
}

// Measurement can be split in two steps, the set-up work that
// applies to the whole line, and the measurement of the actual
// character. Functions like coordsChar, that need to do a lot of
// measurements in a row, can thus ensure that the set-up work is
// only done once.
function prepareMeasureForLine(cm, line) {
  let lineN = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.lineNo)(line)
  let view = findViewForLine(cm, lineN)
  if (view && !view.text) {
    view = null
  } else if (view && view.changes) {
    (0,_display_update_line_js__WEBPACK_IMPORTED_MODULE_10__.updateLineForChanges)(cm, view, lineN, getDimensions(cm))
    cm.curOp.forceUpdate = true
  }
  if (!view)
    view = updateExternalMeasurement(cm, line)

  let info = mapFromLineView(view, line, lineN)
  return {
    line: line, view: view, rect: null,
    map: info.map, cache: info.cache, before: info.before,
    hasHeights: false
  }
}

// Given a prepared measurement object, measures the position of an
// actual character (or fetches it from the cache).
function measureCharPrepared(cm, prepared, ch, bias, varHeight) {
  if (prepared.before) ch = -1
  let key = ch + (bias || ""), found
  if (prepared.cache.hasOwnProperty(key)) {
    found = prepared.cache[key]
  } else {
    if (!prepared.rect)
      prepared.rect = prepared.view.text.getBoundingClientRect()
    if (!prepared.hasHeights) {
      ensureLineHeights(cm, prepared.view, prepared.rect)
      prepared.hasHeights = true
    }
    found = measureCharInner(cm, prepared, ch, bias)
    if (!found.bogus) prepared.cache[key] = found
  }
  return {left: found.left, right: found.right,
          top: varHeight ? found.rtop : found.top,
          bottom: varHeight ? found.rbottom : found.bottom}
}

let nullRect = {left: 0, right: 0, top: 0, bottom: 0}

function nodeAndOffsetInLineMap(map, ch, bias) {
  let node, start, end, collapse, mStart, mEnd
  // First, search the line map for the text node corresponding to,
  // or closest to, the target character.
  for (let i = 0; i < map.length; i += 3) {
    mStart = map[i]
    mEnd = map[i + 1]
    if (ch < mStart) {
      start = 0; end = 1
      collapse = "left"
    } else if (ch < mEnd) {
      start = ch - mStart
      end = start + 1
    } else if (i == map.length - 3 || ch == mEnd && map[i + 3] > ch) {
      end = mEnd - mStart
      start = end - 1
      if (ch >= mEnd) collapse = "right"
    }
    if (start != null) {
      node = map[i + 2]
      if (mStart == mEnd && bias == (node.insertLeft ? "left" : "right"))
        collapse = bias
      if (bias == "left" && start == 0)
        while (i && map[i - 2] == map[i - 3] && map[i - 1].insertLeft) {
          node = map[(i -= 3) + 2]
          collapse = "left"
        }
      if (bias == "right" && start == mEnd - mStart)
        while (i < map.length - 3 && map[i + 3] == map[i + 4] && !map[i + 5].insertLeft) {
          node = map[(i += 3) + 2]
          collapse = "right"
        }
      break
    }
  }
  return {node: node, start: start, end: end, collapse: collapse, coverStart: mStart, coverEnd: mEnd}
}

function getUsefulRect(rects, bias) {
  let rect = nullRect
  if (bias == "left") for (let i = 0; i < rects.length; i++) {
    if ((rect = rects[i]).left != rect.right) break
  } else for (let i = rects.length - 1; i >= 0; i--) {
    if ((rect = rects[i]).left != rect.right) break
  }
  return rect
}

function measureCharInner(cm, prepared, ch, bias) {
  let place = nodeAndOffsetInLineMap(prepared.map, ch, bias)
  let node = place.node, start = place.start, end = place.end, collapse = place.collapse

  let rect
  if (node.nodeType == 3) { // If it is a text node, use a range to retrieve the coordinates.
    for (let i = 0; i < 4; i++) { // Retry a maximum of 4 times when nonsense rectangles are returned
      while (start && (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.isExtendingChar)(prepared.line.text.charAt(place.coverStart + start))) --start
      while (place.coverStart + end < place.coverEnd && (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.isExtendingChar)(prepared.line.text.charAt(place.coverStart + end))) ++end
      if (_util_browser_js__WEBPACK_IMPORTED_MODULE_5__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_5__.ie_version < 9 && start == 0 && end == place.coverEnd - place.coverStart)
        rect = node.parentNode.getBoundingClientRect()
      else
        rect = getUsefulRect((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.range)(node, start, end).getClientRects(), bias)
      if (rect.left || rect.right || start == 0) break
      end = start
      start = start - 1
      collapse = "right"
    }
    if (_util_browser_js__WEBPACK_IMPORTED_MODULE_5__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_5__.ie_version < 11) rect = maybeUpdateRectForZooming(cm.display.measure, rect)
  } else { // If it is a widget, simply get the box for the whole widget.
    if (start > 0) collapse = bias = "right"
    let rects
    if (cm.options.lineWrapping && (rects = node.getClientRects()).length > 1)
      rect = rects[bias == "right" ? rects.length - 1 : 0]
    else
      rect = node.getBoundingClientRect()
  }
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_5__.ie && _util_browser_js__WEBPACK_IMPORTED_MODULE_5__.ie_version < 9 && !start && (!rect || !rect.left && !rect.right)) {
    let rSpan = node.parentNode.getClientRects()[0]
    if (rSpan)
      rect = {left: rSpan.left, right: rSpan.left + charWidth(cm.display), top: rSpan.top, bottom: rSpan.bottom}
    else
      rect = nullRect
  }

  let rtop = rect.top - prepared.rect.top, rbot = rect.bottom - prepared.rect.top
  let mid = (rtop + rbot) / 2
  let heights = prepared.view.measure.heights
  let i = 0
  for (; i < heights.length - 1; i++)
    if (mid < heights[i]) break
  let top = i ? heights[i - 1] : 0, bot = heights[i]
  let result = {left: (collapse == "right" ? rect.right : rect.left) - prepared.rect.left,
                right: (collapse == "left" ? rect.left : rect.right) - prepared.rect.left,
                top: top, bottom: bot}
  if (!rect.left && !rect.right) result.bogus = true
  if (!cm.options.singleCursorHeightPerLine) { result.rtop = rtop; result.rbottom = rbot }

  return result
}

// Work around problem with bounding client rects on ranges being
// returned incorrectly when zoomed on IE10 and below.
function maybeUpdateRectForZooming(measure, rect) {
  if (!window.screen || screen.logicalXDPI == null ||
      screen.logicalXDPI == screen.deviceXDPI || !(0,_util_feature_detection_js__WEBPACK_IMPORTED_MODULE_8__.hasBadZoomedRects)(measure))
    return rect
  let scaleX = screen.logicalXDPI / screen.deviceXDPI
  let scaleY = screen.logicalYDPI / screen.deviceYDPI
  return {left: rect.left * scaleX, right: rect.right * scaleX,
          top: rect.top * scaleY, bottom: rect.bottom * scaleY}
}

function clearLineMeasurementCacheFor(lineView) {
  if (lineView.measure) {
    lineView.measure.cache = {}
    lineView.measure.heights = null
    if (lineView.rest) for (let i = 0; i < lineView.rest.length; i++)
      lineView.measure.caches[i] = {}
  }
}

function clearLineMeasurementCache(cm) {
  cm.display.externalMeasure = null
  ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.removeChildren)(cm.display.lineMeasure)
  for (let i = 0; i < cm.display.view.length; i++)
    clearLineMeasurementCacheFor(cm.display.view[i])
}

function clearCaches(cm) {
  clearLineMeasurementCache(cm)
  cm.display.cachedCharWidth = cm.display.cachedTextHeight = cm.display.cachedPaddingH = null
  if (!cm.options.lineWrapping) cm.display.maxLineChanged = true
  cm.display.lineNumChars = null
}

function pageScrollX() {
  // Work around https://bugs.chromium.org/p/chromium/issues/detail?id=489206
  // which causes page_Offset and bounding client rects to use
  // different reference viewports and invalidate our calculations.
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_5__.chrome && _util_browser_js__WEBPACK_IMPORTED_MODULE_5__.android) return -(document.body.getBoundingClientRect().left - parseInt(getComputedStyle(document.body).marginLeft))
  return window.pageXOffset || (document.documentElement || document.body).scrollLeft
}
function pageScrollY() {
  if (_util_browser_js__WEBPACK_IMPORTED_MODULE_5__.chrome && _util_browser_js__WEBPACK_IMPORTED_MODULE_5__.android) return -(document.body.getBoundingClientRect().top - parseInt(getComputedStyle(document.body).marginTop))
  return window.pageYOffset || (document.documentElement || document.body).scrollTop
}

function widgetTopHeight(lineObj) {
  let height = 0
  if (lineObj.widgets) for (let i = 0; i < lineObj.widgets.length; ++i) if (lineObj.widgets[i].above)
    height += (0,_widgets_js__WEBPACK_IMPORTED_MODULE_11__.widgetHeight)(lineObj.widgets[i])
  return height
}

// Converts a {top, bottom, left, right} box from line-local
// coordinates into another coordinate system. Context may be one of
// "line", "div" (display.lineDiv), "local"./null (editor), "window",
// or "page".
function intoCoordSystem(cm, lineObj, rect, context, includeWidgets) {
  if (!includeWidgets) {
    let height = widgetTopHeight(lineObj)
    rect.top += height; rect.bottom += height
  }
  if (context == "line") return rect
  if (!context) context = "local"
  let yOff = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_2__.heightAtLine)(lineObj)
  if (context == "local") yOff += paddingTop(cm.display)
  else yOff -= cm.display.viewOffset
  if (context == "page" || context == "window") {
    let lOff = cm.display.lineSpace.getBoundingClientRect()
    yOff += lOff.top + (context == "window" ? 0 : pageScrollY())
    let xOff = lOff.left + (context == "window" ? 0 : pageScrollX())
    rect.left += xOff; rect.right += xOff
  }
  rect.top += yOff; rect.bottom += yOff
  return rect
}

// Coverts a box from "div" coords to another coordinate system.
// Context may be "window", "page", "div", or "local"./null.
function fromCoordSystem(cm, coords, context) {
  if (context == "div") return coords
  let left = coords.left, top = coords.top
  // First move into "page" coordinate system
  if (context == "page") {
    left -= pageScrollX()
    top -= pageScrollY()
  } else if (context == "local" || !context) {
    let localBox = cm.display.sizer.getBoundingClientRect()
    left += localBox.left
    top += localBox.top
  }

  let lineSpaceBox = cm.display.lineSpace.getBoundingClientRect()
  return {left: left - lineSpaceBox.left, top: top - lineSpaceBox.top}
}

function charCoords(cm, pos, context, lineObj, bias) {
  if (!lineObj) lineObj = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(cm.doc, pos.line)
  return intoCoordSystem(cm, lineObj, measureChar(cm, lineObj, pos.ch, bias), context)
}

// Returns a box for a given cursor position, which may have an
// 'other' property containing the position of the secondary cursor
// on a bidi boundary.
// A cursor Pos(line, char, "before") is on the same visual line as `char - 1`
// and after `char - 1` in writing order of `char - 1`
// A cursor Pos(line, char, "after") is on the same visual line as `char`
// and before `char` in writing order of `char`
// Examples (upper-case letters are RTL, lower-case are LTR):
//     Pos(0, 1, ...)
//     before   after
// ab     a|b     a|b
// aB     a|B     aB|
// Ab     |Ab     A|b
// AB     B|A     B|A
// Every position after the last character on a line is considered to stick
// to the last character on the line.
function cursorCoords(cm, pos, context, lineObj, preparedMeasure, varHeight) {
  lineObj = lineObj || (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(cm.doc, pos.line)
  if (!preparedMeasure) preparedMeasure = prepareMeasureForLine(cm, lineObj)
  function get(ch, right) {
    let m = measureCharPrepared(cm, preparedMeasure, ch, right ? "right" : "left", varHeight)
    if (right) m.left = m.right; else m.right = m.left
    return intoCoordSystem(cm, lineObj, m, context)
  }
  let order = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_4__.getOrder)(lineObj, cm.doc.direction), ch = pos.ch, sticky = pos.sticky
  if (ch >= lineObj.text.length) {
    ch = lineObj.text.length
    sticky = "before"
  } else if (ch <= 0) {
    ch = 0
    sticky = "after"
  }
  if (!order) return get(sticky == "before" ? ch - 1 : ch, sticky == "before")

  function getBidi(ch, partPos, invert) {
    let part = order[partPos], right = part.level == 1
    return get(invert ? ch - 1 : ch, right != invert)
  }
  let partPos = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_4__.getBidiPartAt)(order, ch, sticky)
  let other = _util_bidi_js__WEBPACK_IMPORTED_MODULE_4__.bidiOther
  let val = getBidi(ch, partPos, sticky == "before")
  if (other != null) val.other = getBidi(ch, other, sticky != "before")
  return val
}

// Used to cheaply estimate the coordinates for a position. Used for
// intermediate scroll updates.
function estimateCoords(cm, pos) {
  let left = 0
  pos = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_1__.clipPos)(cm.doc, pos)
  if (!cm.options.lineWrapping) left = charWidth(cm.display) * pos.ch
  let lineObj = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(cm.doc, pos.line)
  let top = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_2__.heightAtLine)(lineObj) + paddingTop(cm.display)
  return {left: left, right: left, top: top, bottom: top + lineObj.height}
}

// Positions returned by coordsChar contain some extra information.
// xRel is the relative x position of the input coordinates compared
// to the found position (so xRel > 0 means the coordinates are to
// the right of the character position, for example). When outside
// is true, that means the coordinates lie outside the line's
// vertical range.
function PosWithInfo(line, ch, sticky, outside, xRel) {
  let pos = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_1__.Pos)(line, ch, sticky)
  pos.xRel = xRel
  if (outside) pos.outside = outside
  return pos
}

// Compute the character position closest to the given coordinates.
// Input must be lineSpace-local ("div" coordinate system).
function coordsChar(cm, x, y) {
  let doc = cm.doc
  y += cm.display.viewOffset
  if (y < 0) return PosWithInfo(doc.first, 0, null, -1, -1)
  let lineN = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.lineAtHeight)(doc, y), last = doc.first + doc.size - 1
  if (lineN > last)
    return PosWithInfo(doc.first + doc.size - 1, (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, last).text.length, null, 1, 1)
  if (x < 0) x = 0

  let lineObj = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, lineN)
  for (;;) {
    let found = coordsCharInner(cm, lineObj, lineN, x, y)
    let collapsed = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_2__.collapsedSpanAround)(lineObj, found.ch + (found.xRel > 0 || found.outside > 0 ? 1 : 0))
    if (!collapsed) return found
    let rangeEnd = collapsed.find(1)
    if (rangeEnd.line == lineN) return rangeEnd
    lineObj = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, lineN = rangeEnd.line)
  }
}

function wrappedLineExtent(cm, lineObj, preparedMeasure, y) {
  y -= widgetTopHeight(lineObj)
  let end = lineObj.text.length
  let begin = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.findFirst)(ch => measureCharPrepared(cm, preparedMeasure, ch - 1).bottom <= y, end, 0)
  end = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.findFirst)(ch => measureCharPrepared(cm, preparedMeasure, ch).top > y, begin, end)
  return {begin, end}
}

function wrappedLineExtentChar(cm, lineObj, preparedMeasure, target) {
  if (!preparedMeasure) preparedMeasure = prepareMeasureForLine(cm, lineObj)
  let targetTop = intoCoordSystem(cm, lineObj, measureCharPrepared(cm, preparedMeasure, target), "line").top
  return wrappedLineExtent(cm, lineObj, preparedMeasure, targetTop)
}

// Returns true if the given side of a box is after the given
// coordinates, in top-to-bottom, left-to-right order.
function boxIsAfter(box, x, y, left) {
  return box.bottom <= y ? false : box.top > y ? true : (left ? box.left : box.right) > x
}

function coordsCharInner(cm, lineObj, lineNo, x, y) {
  // Move y into line-local coordinate space
  y -= (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_2__.heightAtLine)(lineObj)
  let preparedMeasure = prepareMeasureForLine(cm, lineObj)
  // When directly calling `measureCharPrepared`, we have to adjust
  // for the widgets at this line.
  let widgetHeight = widgetTopHeight(lineObj)
  let begin = 0, end = lineObj.text.length, ltr = true

  let order = (0,_util_bidi_js__WEBPACK_IMPORTED_MODULE_4__.getOrder)(lineObj, cm.doc.direction)
  // If the line isn't plain left-to-right text, first figure out
  // which bidi section the coordinates fall into.
  if (order) {
    let part = (cm.options.lineWrapping ? coordsBidiPartWrapped : coordsBidiPart)
                 (cm, lineObj, lineNo, preparedMeasure, order, x, y)
    ltr = part.level != 1
    // The awkward -1 offsets are needed because findFirst (called
    // on these below) will treat its first bound as inclusive,
    // second as exclusive, but we want to actually address the
    // characters in the part's range
    begin = ltr ? part.from : part.to - 1
    end = ltr ? part.to : part.from - 1
  }

  // A binary search to find the first character whose bounding box
  // starts after the coordinates. If we run across any whose box wrap
  // the coordinates, store that.
  let chAround = null, boxAround = null
  let ch = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.findFirst)(ch => {
    let box = measureCharPrepared(cm, preparedMeasure, ch)
    box.top += widgetHeight; box.bottom += widgetHeight
    if (!boxIsAfter(box, x, y, false)) return false
    if (box.top <= y && box.left <= x) {
      chAround = ch
      boxAround = box
    }
    return true
  }, begin, end)

  let baseX, sticky, outside = false
  // If a box around the coordinates was found, use that
  if (boxAround) {
    // Distinguish coordinates nearer to the left or right side of the box
    let atLeft = x - boxAround.left < boxAround.right - x, atStart = atLeft == ltr
    ch = chAround + (atStart ? 0 : 1)
    sticky = atStart ? "after" : "before"
    baseX = atLeft ? boxAround.left : boxAround.right
  } else {
    // (Adjust for extended bound, if necessary.)
    if (!ltr && (ch == end || ch == begin)) ch++
    // To determine which side to associate with, get the box to the
    // left of the character and compare it's vertical position to the
    // coordinates
    sticky = ch == 0 ? "after" : ch == lineObj.text.length ? "before" :
      (measureCharPrepared(cm, preparedMeasure, ch - (ltr ? 1 : 0)).bottom + widgetHeight <= y) == ltr ?
      "after" : "before"
    // Now get accurate coordinates for this place, in order to get a
    // base X position
    let coords = cursorCoords(cm, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_1__.Pos)(lineNo, ch, sticky), "line", lineObj, preparedMeasure)
    baseX = coords.left
    outside = y < coords.top ? -1 : y >= coords.bottom ? 1 : 0
  }

  ch = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.skipExtendingChars)(lineObj.text, ch, 1)
  return PosWithInfo(lineNo, ch, sticky, outside, x - baseX)
}

function coordsBidiPart(cm, lineObj, lineNo, preparedMeasure, order, x, y) {
  // Bidi parts are sorted left-to-right, and in a non-line-wrapping
  // situation, we can take this ordering to correspond to the visual
  // ordering. This finds the first part whose end is after the given
  // coordinates.
  let index = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.findFirst)(i => {
    let part = order[i], ltr = part.level != 1
    return boxIsAfter(cursorCoords(cm, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_1__.Pos)(lineNo, ltr ? part.to : part.from, ltr ? "before" : "after"),
                                   "line", lineObj, preparedMeasure), x, y, true)
  }, 0, order.length - 1)
  let part = order[index]
  // If this isn't the first part, the part's start is also after
  // the coordinates, and the coordinates aren't on the same line as
  // that start, move one part back.
  if (index > 0) {
    let ltr = part.level != 1
    let start = cursorCoords(cm, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_1__.Pos)(lineNo, ltr ? part.from : part.to, ltr ? "after" : "before"),
                             "line", lineObj, preparedMeasure)
    if (boxIsAfter(start, x, y, true) && start.top > y)
      part = order[index - 1]
  }
  return part
}

function coordsBidiPartWrapped(cm, lineObj, _lineNo, preparedMeasure, order, x, y) {
  // In a wrapped line, rtl text on wrapping boundaries can do things
  // that don't correspond to the ordering in our `order` array at
  // all, so a binary search doesn't work, and we want to return a
  // part that only spans one line so that the binary search in
  // coordsCharInner is safe. As such, we first find the extent of the
  // wrapped line, and then do a flat search in which we discard any
  // spans that aren't on the line.
  let {begin, end} = wrappedLineExtent(cm, lineObj, preparedMeasure, y)
  if (/\s/.test(lineObj.text.charAt(end - 1))) end--
  let part = null, closestDist = null
  for (let i = 0; i < order.length; i++) {
    let p = order[i]
    if (p.from >= end || p.to <= begin) continue
    let ltr = p.level != 1
    let endX = measureCharPrepared(cm, preparedMeasure, ltr ? Math.min(end, p.to) - 1 : Math.max(begin, p.from)).right
    // Weigh against spans ending before this, so that they are only
    // picked if nothing ends after
    let dist = endX < x ? x - endX + 1e9 : endX - x
    if (!part || closestDist > dist) {
      part = p
      closestDist = dist
    }
  }
  if (!part) part = order[order.length - 1]
  // Clip the part to the wrapped line.
  if (part.from < begin) part = {from: begin, to: part.to, level: part.level}
  if (part.to > end) part = {from: part.from, to: end, level: part.level}
  return part
}

let measureText
// Compute the default text height.
function textHeight(display) {
  if (display.cachedTextHeight != null) return display.cachedTextHeight
  if (measureText == null) {
    measureText = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.elt)("pre", null, "CodeMirror-line-like")
    // Measure a bunch of lines, for browsers that compute
    // fractional heights.
    for (let i = 0; i < 49; ++i) {
      measureText.appendChild(document.createTextNode("x"))
      measureText.appendChild((0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.elt)("br"))
    }
    measureText.appendChild(document.createTextNode("x"))
  }
  (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.removeChildrenAndAdd)(display.measure, measureText)
  let height = measureText.offsetHeight / 50
  if (height > 3) display.cachedTextHeight = height
  ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.removeChildren)(display.measure)
  return height || 1
}

// Compute the default character width.
function charWidth(display) {
  if (display.cachedCharWidth != null) return display.cachedCharWidth
  let anchor = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.elt)("span", "xxxxxxxxxx")
  let pre = (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.elt)("pre", [anchor], "CodeMirror-line-like")
  ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_6__.removeChildrenAndAdd)(display.measure, pre)
  let rect = anchor.getBoundingClientRect(), width = (rect.right - rect.left) / 10
  if (width > 2) display.cachedCharWidth = width
  return width || 10
}

// Do a bulk-read of the DOM positions and sizes needed to draw the
// view, so that we don't interleave reading and writing to the DOM.
function getDimensions(cm) {
  let d = cm.display, left = {}, width = {}
  let gutterLeft = d.gutters.clientLeft
  for (let n = d.gutters.firstChild, i = 0; n; n = n.nextSibling, ++i) {
    let id = cm.display.gutterSpecs[i].className
    left[id] = n.offsetLeft + n.clientLeft + gutterLeft
    width[id] = n.clientWidth
  }
  return {fixedPos: compensateForHScroll(d),
          gutterTotalWidth: d.gutters.offsetWidth,
          gutterLeft: left,
          gutterWidth: width,
          wrapperWidth: d.wrapper.clientWidth}
}

// Computes display.scroller.scrollLeft + display.gutters.offsetWidth,
// but using getBoundingClientRect to get a sub-pixel-accurate
// result.
function compensateForHScroll(display) {
  return display.scroller.getBoundingClientRect().left - display.sizer.getBoundingClientRect().left
}

// Returns a function that estimates the height of a line, to use as
// first approximation until the line becomes visible (and is thus
// properly measurable).
function estimateHeight(cm) {
  let th = textHeight(cm.display), wrapping = cm.options.lineWrapping
  let perLine = wrapping && Math.max(5, cm.display.scroller.clientWidth / charWidth(cm.display) - 3)
  return line => {
    if ((0,_line_spans_js__WEBPACK_IMPORTED_MODULE_2__.lineIsHidden)(cm.doc, line)) return 0

    let widgetsHeight = 0
    if (line.widgets) for (let i = 0; i < line.widgets.length; i++) {
      if (line.widgets[i].height) widgetsHeight += line.widgets[i].height
    }

    if (wrapping)
      return widgetsHeight + (Math.ceil(line.text.length / perLine) || 1) * th
    else
      return widgetsHeight + th
  }
}

function estimateLineHeights(cm) {
  let doc = cm.doc, est = estimateHeight(cm)
  doc.iter(line => {
    let estHeight = est(line)
    if (estHeight != line.height) (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.updateLineHeight)(line, estHeight)
  })
}

// Given a mouse event, find the corresponding position. If liberal
// is false, it checks whether a gutter or scrollbar was clicked,
// and returns null if it was. forRect is used by rectangular
// selections, and tries to estimate a character position even for
// coordinates beyond the right of the text.
function posFromMouse(cm, e, liberal, forRect) {
  let display = cm.display
  if (!liberal && (0,_util_event_js__WEBPACK_IMPORTED_MODULE_7__.e_target)(e).getAttribute("cm-not-content") == "true") return null

  let x, y, space = display.lineSpace.getBoundingClientRect()
  // Fails unpredictably on IE[67] when mouse is dragged around quickly.
  try { x = e.clientX - space.left; y = e.clientY - space.top }
  catch (e) { return null }
  let coords = coordsChar(cm, x, y), line
  if (forRect && coords.xRel > 0 && (line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(cm.doc, coords.line).text).length == coords.ch) {
    let colDiff = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_9__.countColumn)(line, line.length, cm.options.tabSize) - line.length
    coords = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_1__.Pos)(coords.line, Math.max(0, Math.round((x - paddingH(cm.display).left) / charWidth(cm.display)) - colDiff))
  }
  return coords
}

// Find the view element corresponding to a given line. Return null
// when the line isn't visible.
function findViewIndex(cm, n) {
  if (n >= cm.display.viewTo) return null
  n -= cm.display.viewFrom
  if (n < 0) return null
  let view = cm.display.view
  for (let i = 0; i < view.length; i++) {
    n -= view[i].size
    if (n < 0) return i
  }
}


/***/ }),

/***/ "./node_modules/codemirror/src/measurement/widgets.js":
/*!************************************************************!*\
  !*** ./node_modules/codemirror/src/measurement/widgets.js ***!
  \************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "widgetHeight": () => (/* binding */ widgetHeight),
/* harmony export */   "eventInWidget": () => (/* binding */ eventInWidget)
/* harmony export */ });
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");



function widgetHeight(widget) {
  if (widget.height != null) return widget.height
  let cm = widget.doc.cm
  if (!cm) return 0
  if (!(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_0__.contains)(document.body, widget.node)) {
    let parentStyle = "position: relative;"
    if (widget.coverGutter)
      parentStyle += "margin-left: -" + cm.display.gutters.offsetWidth + "px;"
    if (widget.noHScroll)
      parentStyle += "width: " + cm.display.wrapper.clientWidth + "px;"
    ;(0,_util_dom_js__WEBPACK_IMPORTED_MODULE_0__.removeChildrenAndAdd)(cm.display.measure, (0,_util_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("div", [widget.node], null, parentStyle))
  }
  return widget.height = widget.node.parentNode.offsetHeight
}

// Return true when the given mouse event happened in a widget
function eventInWidget(display, e) {
  for (let n = (0,_util_event_js__WEBPACK_IMPORTED_MODULE_1__.e_target)(e); n != display.wrapper; n = n.parentNode) {
    if (!n || (n.nodeType == 1 && n.getAttribute("cm-ignore-events") == "true") ||
        (n.parentNode == display.sizer && n != display.mover))
      return true
  }
}


/***/ }),

/***/ "./node_modules/codemirror/src/model/change_measurement.js":
/*!*****************************************************************!*\
  !*** ./node_modules/codemirror/src/model/change_measurement.js ***!
  \*****************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "changeEnd": () => (/* binding */ changeEnd),
/* harmony export */   "computeSelAfterChange": () => (/* binding */ computeSelAfterChange),
/* harmony export */   "computeReplacedSel": () => (/* binding */ computeReplacedSel)
/* harmony export */ });
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _selection_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ./selection.js */ "./node_modules/codemirror/src/model/selection.js");





// Compute the position of the end of a change (its 'to' property
// refers to the pre-change end).
function changeEnd(change) {
  if (!change.text) return change.to
  return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos)(change.from.line + change.text.length - 1,
             (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_1__.lst)(change.text).length + (change.text.length == 1 ? change.from.ch : 0))
}

// Adjust a position to refer to the post-change position of the
// same text, or the end of the change if the change covers it.
function adjustForChange(pos, change) {
  if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.cmp)(pos, change.from) < 0) return pos
  if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.cmp)(pos, change.to) <= 0) return changeEnd(change)

  let line = pos.line + change.text.length - (change.to.line - change.from.line) - 1, ch = pos.ch
  if (pos.line == change.to.line) ch += changeEnd(change).ch - change.to.ch
  return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos)(line, ch)
}

function computeSelAfterChange(doc, change) {
  let out = []
  for (let i = 0; i < doc.sel.ranges.length; i++) {
    let range = doc.sel.ranges[i]
    out.push(new _selection_js__WEBPACK_IMPORTED_MODULE_2__.Range(adjustForChange(range.anchor, change),
                       adjustForChange(range.head, change)))
  }
  return (0,_selection_js__WEBPACK_IMPORTED_MODULE_2__.normalizeSelection)(doc.cm, out, doc.sel.primIndex)
}

function offsetPos(pos, old, nw) {
  if (pos.line == old.line)
    return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos)(nw.line, pos.ch - old.ch + nw.ch)
  else
    return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos)(nw.line + (pos.line - old.line), pos.ch)
}

// Used by replaceSelections to allow moving the selection to the
// start or around the replaced test. Hint may be "start" or "around".
function computeReplacedSel(doc, changes, hint) {
  let out = []
  let oldPrev = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.Pos)(doc.first, 0), newPrev = oldPrev
  for (let i = 0; i < changes.length; i++) {
    let change = changes[i]
    let from = offsetPos(change.from, oldPrev, newPrev)
    let to = offsetPos(changeEnd(change), oldPrev, newPrev)
    oldPrev = change.to
    newPrev = to
    if (hint == "around") {
      let range = doc.sel.ranges[i], inv = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.cmp)(range.head, range.anchor) < 0
      out[i] = new _selection_js__WEBPACK_IMPORTED_MODULE_2__.Range(inv ? to : from, inv ? from : to)
    } else {
      out[i] = new _selection_js__WEBPACK_IMPORTED_MODULE_2__.Range(from, from)
    }
  }
  return new _selection_js__WEBPACK_IMPORTED_MODULE_2__.Selection(out, doc.sel.primIndex)
}


/***/ }),

/***/ "./node_modules/codemirror/src/model/changes.js":
/*!******************************************************!*\
  !*** ./node_modules/codemirror/src/model/changes.js ***!
  \******************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "makeChange": () => (/* binding */ makeChange),
/* harmony export */   "makeChangeFromHistory": () => (/* binding */ makeChangeFromHistory),
/* harmony export */   "replaceRange": () => (/* binding */ replaceRange),
/* harmony export */   "changeLine": () => (/* binding */ changeLine)
/* harmony export */ });
/* harmony import */ var _line_highlight_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/highlight.js */ "./node_modules/codemirror/src/line/highlight.js");
/* harmony import */ var _display_highlight_worker_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../display/highlight_worker.js */ "./node_modules/codemirror/src/display/highlight_worker.js");
/* harmony import */ var _display_operations_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../display/operations.js */ "./node_modules/codemirror/src/display/operations.js");
/* harmony import */ var _display_view_tracking_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../display/view_tracking.js */ "./node_modules/codemirror/src/display/view_tracking.js");
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_saw_special_spans_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../line/saw_special_spans.js */ "./node_modules/codemirror/src/line/saw_special_spans.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_10__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _util_operation_group_js__WEBPACK_IMPORTED_MODULE_11__ = __webpack_require__(/*! ../util/operation_group.js */ "./node_modules/codemirror/src/util/operation_group.js");
/* harmony import */ var _change_measurement_js__WEBPACK_IMPORTED_MODULE_12__ = __webpack_require__(/*! ./change_measurement.js */ "./node_modules/codemirror/src/model/change_measurement.js");
/* harmony import */ var _document_data_js__WEBPACK_IMPORTED_MODULE_13__ = __webpack_require__(/*! ./document_data.js */ "./node_modules/codemirror/src/model/document_data.js");
/* harmony import */ var _history_js__WEBPACK_IMPORTED_MODULE_14__ = __webpack_require__(/*! ./history.js */ "./node_modules/codemirror/src/model/history.js");
/* harmony import */ var _selection_js__WEBPACK_IMPORTED_MODULE_15__ = __webpack_require__(/*! ./selection.js */ "./node_modules/codemirror/src/model/selection.js");
/* harmony import */ var _selection_updates_js__WEBPACK_IMPORTED_MODULE_16__ = __webpack_require__(/*! ./selection_updates.js */ "./node_modules/codemirror/src/model/selection_updates.js");



















// UPDATING

// Allow "beforeChange" event handlers to influence a change
function filterChange(doc, change, update) {
  let obj = {
    canceled: false,
    from: change.from,
    to: change.to,
    text: change.text,
    origin: change.origin,
    cancel: () => obj.canceled = true
  }
  if (update) obj.update = (from, to, text, origin) => {
    if (from) obj.from = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.clipPos)(doc, from)
    if (to) obj.to = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.clipPos)(doc, to)
    if (text) obj.text = text
    if (origin !== undefined) obj.origin = origin
  }
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_9__.signal)(doc, "beforeChange", doc, obj)
  if (doc.cm) (0,_util_event_js__WEBPACK_IMPORTED_MODULE_9__.signal)(doc.cm, "beforeChange", doc.cm, obj)

  if (obj.canceled) {
    if (doc.cm) doc.cm.curOp.updateInput = 2
    return null
  }
  return {from: obj.from, to: obj.to, text: obj.text, origin: obj.origin}
}

// Apply a change to a document, and add it to the document's
// history, and propagating it to all linked documents.
function makeChange(doc, change, ignoreReadOnly) {
  if (doc.cm) {
    if (!doc.cm.curOp) return (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_2__.operation)(doc.cm, makeChange)(doc, change, ignoreReadOnly)
    if (doc.cm.state.suppressEdits) return
  }

  if ((0,_util_event_js__WEBPACK_IMPORTED_MODULE_9__.hasHandler)(doc, "beforeChange") || doc.cm && (0,_util_event_js__WEBPACK_IMPORTED_MODULE_9__.hasHandler)(doc.cm, "beforeChange")) {
    change = filterChange(doc, change, true)
    if (!change) return
  }

  // Possibly split or suppress the update based on the presence
  // of read-only spans in its range.
  let split = _line_saw_special_spans_js__WEBPACK_IMPORTED_MODULE_5__.sawReadOnlySpans && !ignoreReadOnly && (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_6__.removeReadOnlyRanges)(doc, change.from, change.to)
  if (split) {
    for (let i = split.length - 1; i >= 0; --i)
      makeChangeInner(doc, {from: split[i].from, to: split[i].to, text: i ? [""] : change.text, origin: change.origin})
  } else {
    makeChangeInner(doc, change)
  }
}

function makeChangeInner(doc, change) {
  if (change.text.length == 1 && change.text[0] == "" && (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.cmp)(change.from, change.to) == 0) return
  let selAfter = (0,_change_measurement_js__WEBPACK_IMPORTED_MODULE_12__.computeSelAfterChange)(doc, change)
  ;(0,_history_js__WEBPACK_IMPORTED_MODULE_14__.addChangeToHistory)(doc, change, selAfter, doc.cm ? doc.cm.curOp.id : NaN)

  makeChangeSingleDoc(doc, change, selAfter, (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_6__.stretchSpansOverChange)(doc, change))
  let rebased = []

  ;(0,_document_data_js__WEBPACK_IMPORTED_MODULE_13__.linkedDocs)(doc, (doc, sharedHist) => {
    if (!sharedHist && (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_10__.indexOf)(rebased, doc.history) == -1) {
      rebaseHist(doc.history, change)
      rebased.push(doc.history)
    }
    makeChangeSingleDoc(doc, change, null, (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_6__.stretchSpansOverChange)(doc, change))
  })
}

// Revert a change stored in a document's history.
function makeChangeFromHistory(doc, type, allowSelectionOnly) {
  let suppress = doc.cm && doc.cm.state.suppressEdits
  if (suppress && !allowSelectionOnly) return

  let hist = doc.history, event, selAfter = doc.sel
  let source = type == "undo" ? hist.done : hist.undone, dest = type == "undo" ? hist.undone : hist.done

  // Verify that there is a useable event (so that ctrl-z won't
  // needlessly clear selection events)
  let i = 0
  for (; i < source.length; i++) {
    event = source[i]
    if (allowSelectionOnly ? event.ranges && !event.equals(doc.sel) : !event.ranges)
      break
  }
  if (i == source.length) return
  hist.lastOrigin = hist.lastSelOrigin = null

  for (;;) {
    event = source.pop()
    if (event.ranges) {
      (0,_history_js__WEBPACK_IMPORTED_MODULE_14__.pushSelectionToHistory)(event, dest)
      if (allowSelectionOnly && !event.equals(doc.sel)) {
        (0,_selection_updates_js__WEBPACK_IMPORTED_MODULE_16__.setSelection)(doc, event, {clearRedo: false})
        return
      }
      selAfter = event
    } else if (suppress) {
      source.push(event)
      return
    } else break
  }

  // Build up a reverse change object to add to the opposite history
  // stack (redo when undoing, and vice versa).
  let antiChanges = []
  ;(0,_history_js__WEBPACK_IMPORTED_MODULE_14__.pushSelectionToHistory)(selAfter, dest)
  dest.push({changes: antiChanges, generation: hist.generation})
  hist.generation = event.generation || ++hist.maxGeneration

  let filter = (0,_util_event_js__WEBPACK_IMPORTED_MODULE_9__.hasHandler)(doc, "beforeChange") || doc.cm && (0,_util_event_js__WEBPACK_IMPORTED_MODULE_9__.hasHandler)(doc.cm, "beforeChange")

  for (let i = event.changes.length - 1; i >= 0; --i) {
    let change = event.changes[i]
    change.origin = type
    if (filter && !filterChange(doc, change, false)) {
      source.length = 0
      return
    }

    antiChanges.push((0,_history_js__WEBPACK_IMPORTED_MODULE_14__.historyChangeFromChange)(doc, change))

    let after = i ? (0,_change_measurement_js__WEBPACK_IMPORTED_MODULE_12__.computeSelAfterChange)(doc, change) : (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_10__.lst)(source)
    makeChangeSingleDoc(doc, change, after, (0,_history_js__WEBPACK_IMPORTED_MODULE_14__.mergeOldSpans)(doc, change))
    if (!i && doc.cm) doc.cm.scrollIntoView({from: change.from, to: (0,_change_measurement_js__WEBPACK_IMPORTED_MODULE_12__.changeEnd)(change)})
    let rebased = []

    // Propagate to the linked documents
    ;(0,_document_data_js__WEBPACK_IMPORTED_MODULE_13__.linkedDocs)(doc, (doc, sharedHist) => {
      if (!sharedHist && (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_10__.indexOf)(rebased, doc.history) == -1) {
        rebaseHist(doc.history, change)
        rebased.push(doc.history)
      }
      makeChangeSingleDoc(doc, change, null, (0,_history_js__WEBPACK_IMPORTED_MODULE_14__.mergeOldSpans)(doc, change))
    })
  }
}

// Sub-views need their line numbers shifted when text is added
// above or below them in the parent document.
function shiftDoc(doc, distance) {
  if (distance == 0) return
  doc.first += distance
  doc.sel = new _selection_js__WEBPACK_IMPORTED_MODULE_15__.Selection((0,_util_misc_js__WEBPACK_IMPORTED_MODULE_10__.map)(doc.sel.ranges, range => new _selection_js__WEBPACK_IMPORTED_MODULE_15__.Range(
    (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(range.anchor.line + distance, range.anchor.ch),
    (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(range.head.line + distance, range.head.ch)
  )), doc.sel.primIndex)
  if (doc.cm) {
    (0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_3__.regChange)(doc.cm, doc.first, doc.first - distance, distance)
    for (let d = doc.cm.display, l = d.viewFrom; l < d.viewTo; l++)
      (0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_3__.regLineChange)(doc.cm, l, "gutter")
  }
}

// More lower-level change function, handling only a single document
// (not linked ones).
function makeChangeSingleDoc(doc, change, selAfter, spans) {
  if (doc.cm && !doc.cm.curOp)
    return (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_2__.operation)(doc.cm, makeChangeSingleDoc)(doc, change, selAfter, spans)

  if (change.to.line < doc.first) {
    shiftDoc(doc, change.text.length - 1 - (change.to.line - change.from.line))
    return
  }
  if (change.from.line > doc.lastLine()) return

  // Clip the change to the size of this doc
  if (change.from.line < doc.first) {
    let shift = change.text.length - 1 - (doc.first - change.from.line)
    shiftDoc(doc, shift)
    change = {from: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(doc.first, 0), to: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(change.to.line + shift, change.to.ch),
              text: [(0,_util_misc_js__WEBPACK_IMPORTED_MODULE_10__.lst)(change.text)], origin: change.origin}
  }
  let last = doc.lastLine()
  if (change.to.line > last) {
    change = {from: change.from, to: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(last, (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_7__.getLine)(doc, last).text.length),
              text: [change.text[0]], origin: change.origin}
  }

  change.removed = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_7__.getBetween)(doc, change.from, change.to)

  if (!selAfter) selAfter = (0,_change_measurement_js__WEBPACK_IMPORTED_MODULE_12__.computeSelAfterChange)(doc, change)
  if (doc.cm) makeChangeSingleDocInEditor(doc.cm, change, spans)
  else (0,_document_data_js__WEBPACK_IMPORTED_MODULE_13__.updateDoc)(doc, change, spans)
  ;(0,_selection_updates_js__WEBPACK_IMPORTED_MODULE_16__.setSelectionNoUndo)(doc, selAfter, _util_misc_js__WEBPACK_IMPORTED_MODULE_10__.sel_dontScroll)

  if (doc.cantEdit && (0,_selection_updates_js__WEBPACK_IMPORTED_MODULE_16__.skipAtomic)(doc, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(doc.firstLine(), 0)))
    doc.cantEdit = false
}

// Handle the interaction of a change to a document with the editor
// that this document is part of.
function makeChangeSingleDocInEditor(cm, change, spans) {
  let doc = cm.doc, display = cm.display, from = change.from, to = change.to

  let recomputeMaxLength = false, checkWidthStart = from.line
  if (!cm.options.lineWrapping) {
    checkWidthStart = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_7__.lineNo)((0,_line_spans_js__WEBPACK_IMPORTED_MODULE_6__.visualLine)((0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_7__.getLine)(doc, from.line)))
    doc.iter(checkWidthStart, to.line + 1, line => {
      if (line == display.maxLine) {
        recomputeMaxLength = true
        return true
      }
    })
  }

  if (doc.sel.contains(change.from, change.to) > -1)
    (0,_util_event_js__WEBPACK_IMPORTED_MODULE_9__.signalCursorActivity)(cm)

  ;(0,_document_data_js__WEBPACK_IMPORTED_MODULE_13__.updateDoc)(doc, change, spans, (0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_8__.estimateHeight)(cm))

  if (!cm.options.lineWrapping) {
    doc.iter(checkWidthStart, from.line + change.text.length, line => {
      let len = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_6__.lineLength)(line)
      if (len > display.maxLineLength) {
        display.maxLine = line
        display.maxLineLength = len
        display.maxLineChanged = true
        recomputeMaxLength = false
      }
    })
    if (recomputeMaxLength) cm.curOp.updateMaxLine = true
  }

  (0,_line_highlight_js__WEBPACK_IMPORTED_MODULE_0__.retreatFrontier)(doc, from.line)
  ;(0,_display_highlight_worker_js__WEBPACK_IMPORTED_MODULE_1__.startWorker)(cm, 400)

  let lendiff = change.text.length - (to.line - from.line) - 1
  // Remember that these lines changed, for updating the display
  if (change.full)
    (0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_3__.regChange)(cm)
  else if (from.line == to.line && change.text.length == 1 && !(0,_document_data_js__WEBPACK_IMPORTED_MODULE_13__.isWholeLineUpdate)(cm.doc, change))
    (0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_3__.regLineChange)(cm, from.line, "text")
  else
    (0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_3__.regChange)(cm, from.line, to.line + 1, lendiff)

  let changesHandler = (0,_util_event_js__WEBPACK_IMPORTED_MODULE_9__.hasHandler)(cm, "changes"), changeHandler = (0,_util_event_js__WEBPACK_IMPORTED_MODULE_9__.hasHandler)(cm, "change")
  if (changeHandler || changesHandler) {
    let obj = {
      from: from, to: to,
      text: change.text,
      removed: change.removed,
      origin: change.origin
    }
    if (changeHandler) (0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_11__.signalLater)(cm, "change", cm, obj)
    if (changesHandler) (cm.curOp.changeObjs || (cm.curOp.changeObjs = [])).push(obj)
  }
  cm.display.selForContextMenu = null
}

function replaceRange(doc, code, from, to, origin) {
  if (!to) to = from
  if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.cmp)(to, from) < 0) [from, to] = [to, from]
  if (typeof code == "string") code = doc.splitLines(code)
  makeChange(doc, {from, to, text: code, origin})
}

// Rebasing/resetting history to deal with externally-sourced changes

function rebaseHistSelSingle(pos, from, to, diff) {
  if (to < pos.line) {
    pos.line += diff
  } else if (from < pos.line) {
    pos.line = from
    pos.ch = 0
  }
}

// Tries to rebase an array of history events given a change in the
// document. If the change touches the same lines as the event, the
// event, and everything 'behind' it, is discarded. If the change is
// before the event, the event's positions are updated. Uses a
// copy-on-write scheme for the positions, to avoid having to
// reallocate them all on every rebase, but also avoid problems with
// shared position objects being unsafely updated.
function rebaseHistArray(array, from, to, diff) {
  for (let i = 0; i < array.length; ++i) {
    let sub = array[i], ok = true
    if (sub.ranges) {
      if (!sub.copied) { sub = array[i] = sub.deepCopy(); sub.copied = true }
      for (let j = 0; j < sub.ranges.length; j++) {
        rebaseHistSelSingle(sub.ranges[j].anchor, from, to, diff)
        rebaseHistSelSingle(sub.ranges[j].head, from, to, diff)
      }
      continue
    }
    for (let j = 0; j < sub.changes.length; ++j) {
      let cur = sub.changes[j]
      if (to < cur.from.line) {
        cur.from = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(cur.from.line + diff, cur.from.ch)
        cur.to = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.Pos)(cur.to.line + diff, cur.to.ch)
      } else if (from <= cur.to.line) {
        ok = false
        break
      }
    }
    if (!ok) {
      array.splice(0, i + 1)
      i = 0
    }
  }
}

function rebaseHist(hist, change) {
  let from = change.from.line, to = change.to.line, diff = change.text.length - (to - from) - 1
  rebaseHistArray(hist.done, from, to, diff)
  rebaseHistArray(hist.undone, from, to, diff)
}

// Utility for applying a change to a line by handle or number,
// returning the number and optionally registering the line as
// changed.
function changeLine(doc, handle, changeType, op) {
  let no = handle, line = handle
  if (typeof handle == "number") line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_7__.getLine)(doc, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_4__.clipLine)(doc, handle))
  else no = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_7__.lineNo)(handle)
  if (no == null) return null
  if (op(line, no) && doc.cm) (0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_3__.regLineChange)(doc.cm, no, changeType)
  return line
}


/***/ }),

/***/ "./node_modules/codemirror/src/model/document_data.js":
/*!************************************************************!*\
  !*** ./node_modules/codemirror/src/model/document_data.js ***!
  \************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "isWholeLineUpdate": () => (/* binding */ isWholeLineUpdate),
/* harmony export */   "updateDoc": () => (/* binding */ updateDoc),
/* harmony export */   "linkedDocs": () => (/* binding */ linkedDocs),
/* harmony export */   "attachDoc": () => (/* binding */ attachDoc),
/* harmony export */   "directionChanged": () => (/* binding */ directionChanged)
/* harmony export */ });
/* harmony import */ var _display_mode_state_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../display/mode_state.js */ "./node_modules/codemirror/src/display/mode_state.js");
/* harmony import */ var _display_operations_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../display/operations.js */ "./node_modules/codemirror/src/display/operations.js");
/* harmony import */ var _display_view_tracking_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../display/view_tracking.js */ "./node_modules/codemirror/src/display/view_tracking.js");
/* harmony import */ var _line_line_data_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../line/line_data.js */ "./node_modules/codemirror/src/line/line_data.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ../measurement/position_measurement.js */ "./node_modules/codemirror/src/measurement/position_measurement.js");
/* harmony import */ var _util_dom_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ../util/dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_8__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _util_operation_group_js__WEBPACK_IMPORTED_MODULE_9__ = __webpack_require__(/*! ../util/operation_group.js */ "./node_modules/codemirror/src/util/operation_group.js");











// DOCUMENT DATA STRUCTURE

// By default, updates that start and end at the beginning of a line
// are treated specially, in order to make the association of line
// widgets and marker elements with the text behave more intuitive.
function isWholeLineUpdate(doc, change) {
  return change.from.ch == 0 && change.to.ch == 0 && (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_8__.lst)(change.text) == "" &&
    (!doc.cm || doc.cm.options.wholeLineUpdateBefore)
}

// Perform a change on the document data structure.
function updateDoc(doc, change, markedSpans, estimateHeight) {
  function spansFor(n) {return markedSpans ? markedSpans[n] : null}
  function update(line, text, spans) {
    (0,_line_line_data_js__WEBPACK_IMPORTED_MODULE_3__.updateLine)(line, text, spans, estimateHeight)
    ;(0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_9__.signalLater)(line, "change", line, change)
  }
  function linesFor(start, end) {
    let result = []
    for (let i = start; i < end; ++i)
      result.push(new _line_line_data_js__WEBPACK_IMPORTED_MODULE_3__.Line(text[i], spansFor(i), estimateHeight))
    return result
  }

  let from = change.from, to = change.to, text = change.text
  let firstLine = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_5__.getLine)(doc, from.line), lastLine = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_5__.getLine)(doc, to.line)
  let lastText = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_8__.lst)(text), lastSpans = spansFor(text.length - 1), nlines = to.line - from.line

  // Adjust the line structure
  if (change.full) {
    doc.insert(0, linesFor(0, text.length))
    doc.remove(text.length, doc.size - text.length)
  } else if (isWholeLineUpdate(doc, change)) {
    // This is a whole-line replace. Treated specially to make
    // sure line objects move the way they are supposed to.
    let added = linesFor(0, text.length - 1)
    update(lastLine, lastLine.text, lastSpans)
    if (nlines) doc.remove(from.line, nlines)
    if (added.length) doc.insert(from.line, added)
  } else if (firstLine == lastLine) {
    if (text.length == 1) {
      update(firstLine, firstLine.text.slice(0, from.ch) + lastText + firstLine.text.slice(to.ch), lastSpans)
    } else {
      let added = linesFor(1, text.length - 1)
      added.push(new _line_line_data_js__WEBPACK_IMPORTED_MODULE_3__.Line(lastText + firstLine.text.slice(to.ch), lastSpans, estimateHeight))
      update(firstLine, firstLine.text.slice(0, from.ch) + text[0], spansFor(0))
      doc.insert(from.line + 1, added)
    }
  } else if (text.length == 1) {
    update(firstLine, firstLine.text.slice(0, from.ch) + text[0] + lastLine.text.slice(to.ch), spansFor(0))
    doc.remove(from.line + 1, nlines)
  } else {
    update(firstLine, firstLine.text.slice(0, from.ch) + text[0], spansFor(0))
    update(lastLine, lastText + lastLine.text.slice(to.ch), lastSpans)
    let added = linesFor(1, text.length - 1)
    if (nlines > 1) doc.remove(from.line + 1, nlines - 1)
    doc.insert(from.line + 1, added)
  }

  (0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_9__.signalLater)(doc, "change", doc, change)
}

// Call f for all linked documents.
function linkedDocs(doc, f, sharedHistOnly) {
  function propagate(doc, skip, sharedHist) {
    if (doc.linked) for (let i = 0; i < doc.linked.length; ++i) {
      let rel = doc.linked[i]
      if (rel.doc == skip) continue
      let shared = sharedHist && rel.sharedHist
      if (sharedHistOnly && !shared) continue
      f(rel.doc, shared)
      propagate(rel.doc, doc, shared)
    }
  }
  propagate(doc, null, true)
}

// Attach a document to an editor.
function attachDoc(cm, doc) {
  if (doc.cm) throw new Error("This document is already in use.")
  cm.doc = doc
  doc.cm = cm
  ;(0,_measurement_position_measurement_js__WEBPACK_IMPORTED_MODULE_6__.estimateLineHeights)(cm)
  ;(0,_display_mode_state_js__WEBPACK_IMPORTED_MODULE_0__.loadMode)(cm)
  setDirectionClass(cm)
  cm.options.direction = doc.direction
  if (!cm.options.lineWrapping) (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_4__.findMaxLine)(cm)
  cm.options.mode = doc.modeOption
  ;(0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_2__.regChange)(cm)
}

function setDirectionClass(cm) {
  ;(cm.doc.direction == "rtl" ? _util_dom_js__WEBPACK_IMPORTED_MODULE_7__.addClass : _util_dom_js__WEBPACK_IMPORTED_MODULE_7__.rmClass)(cm.display.lineDiv, "CodeMirror-rtl")
}

function directionChanged(cm) {
  (0,_display_operations_js__WEBPACK_IMPORTED_MODULE_1__.runInOp)(cm, () => {
    setDirectionClass(cm)
    ;(0,_display_view_tracking_js__WEBPACK_IMPORTED_MODULE_2__.regChange)(cm)
  })
}


/***/ }),

/***/ "./node_modules/codemirror/src/model/history.js":
/*!******************************************************!*\
  !*** ./node_modules/codemirror/src/model/history.js ***!
  \******************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "History": () => (/* binding */ History),
/* harmony export */   "historyChangeFromChange": () => (/* binding */ historyChangeFromChange),
/* harmony export */   "addChangeToHistory": () => (/* binding */ addChangeToHistory),
/* harmony export */   "addSelectionToHistory": () => (/* binding */ addSelectionToHistory),
/* harmony export */   "pushSelectionToHistory": () => (/* binding */ pushSelectionToHistory),
/* harmony export */   "mergeOldSpans": () => (/* binding */ mergeOldSpans),
/* harmony export */   "copyHistoryArray": () => (/* binding */ copyHistoryArray)
/* harmony export */ });
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_spans_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../line/spans.js */ "./node_modules/codemirror/src/line/spans.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _change_measurement_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ./change_measurement.js */ "./node_modules/codemirror/src/model/change_measurement.js");
/* harmony import */ var _document_data_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ./document_data.js */ "./node_modules/codemirror/src/model/document_data.js");
/* harmony import */ var _selection_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ./selection.js */ "./node_modules/codemirror/src/model/selection.js");










function History(prev) {
  // Arrays of change events and selections. Doing something adds an
  // event to done and clears undo. Undoing moves events from done
  // to undone, redoing moves them in the other direction.
  this.done = []; this.undone = []
  this.undoDepth = prev ? prev.undoDepth : Infinity
  // Used to track when changes can be merged into a single undo
  // event
  this.lastModTime = this.lastSelTime = 0
  this.lastOp = this.lastSelOp = null
  this.lastOrigin = this.lastSelOrigin = null
  // Used by the isClean() method
  this.generation = this.maxGeneration = prev ? prev.maxGeneration : 1
}

// Create a history change event from an updateDoc-style change
// object.
function historyChangeFromChange(doc, change) {
  let histChange = {from: (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.copyPos)(change.from), to: (0,_change_measurement_js__WEBPACK_IMPORTED_MODULE_5__.changeEnd)(change), text: (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_2__.getBetween)(doc, change.from, change.to)}
  attachLocalSpans(doc, histChange, change.from.line, change.to.line + 1)
  ;(0,_document_data_js__WEBPACK_IMPORTED_MODULE_6__.linkedDocs)(doc, doc => attachLocalSpans(doc, histChange, change.from.line, change.to.line + 1), true)
  return histChange
}

// Pop all selection events off the end of a history array. Stop at
// a change event.
function clearSelectionEvents(array) {
  while (array.length) {
    let last = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(array)
    if (last.ranges) array.pop()
    else break
  }
}

// Find the top change event in the history. Pop off selection
// events that are in the way.
function lastChangeEvent(hist, force) {
  if (force) {
    clearSelectionEvents(hist.done)
    return (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(hist.done)
  } else if (hist.done.length && !(0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(hist.done).ranges) {
    return (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(hist.done)
  } else if (hist.done.length > 1 && !hist.done[hist.done.length - 2].ranges) {
    hist.done.pop()
    return (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(hist.done)
  }
}

// Register a change in the history. Merges changes that are within
// a single operation, or are close together with an origin that
// allows merging (starting with "+") into a single event.
function addChangeToHistory(doc, change, selAfter, opId) {
  let hist = doc.history
  hist.undone.length = 0
  let time = +new Date, cur
  let last

  if ((hist.lastOp == opId ||
       hist.lastOrigin == change.origin && change.origin &&
       ((change.origin.charAt(0) == "+" && hist.lastModTime > time - (doc.cm ? doc.cm.options.historyEventDelay : 500)) ||
        change.origin.charAt(0) == "*")) &&
      (cur = lastChangeEvent(hist, hist.lastOp == opId))) {
    // Merge this change into the last event
    last = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(cur.changes)
    if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.cmp)(change.from, change.to) == 0 && (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.cmp)(change.from, last.to) == 0) {
      // Optimized case for simple insertion -- don't want to add
      // new changesets for every character typed
      last.to = (0,_change_measurement_js__WEBPACK_IMPORTED_MODULE_5__.changeEnd)(change)
    } else {
      // Add new sub-event
      cur.changes.push(historyChangeFromChange(doc, change))
    }
  } else {
    // Can not be merged, start a new event.
    let before = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(hist.done)
    if (!before || !before.ranges)
      pushSelectionToHistory(doc.sel, hist.done)
    cur = {changes: [historyChangeFromChange(doc, change)],
           generation: hist.generation}
    hist.done.push(cur)
    while (hist.done.length > hist.undoDepth) {
      hist.done.shift()
      if (!hist.done[0].ranges) hist.done.shift()
    }
  }
  hist.done.push(selAfter)
  hist.generation = ++hist.maxGeneration
  hist.lastModTime = hist.lastSelTime = time
  hist.lastOp = hist.lastSelOp = opId
  hist.lastOrigin = hist.lastSelOrigin = change.origin

  if (!last) (0,_util_event_js__WEBPACK_IMPORTED_MODULE_3__.signal)(doc, "historyAdded")
}

function selectionEventCanBeMerged(doc, origin, prev, sel) {
  let ch = origin.charAt(0)
  return ch == "*" ||
    ch == "+" &&
    prev.ranges.length == sel.ranges.length &&
    prev.somethingSelected() == sel.somethingSelected() &&
    new Date - doc.history.lastSelTime <= (doc.cm ? doc.cm.options.historyEventDelay : 500)
}

// Called whenever the selection changes, sets the new selection as
// the pending selection in the history, and pushes the old pending
// selection into the 'done' array when it was significantly
// different (in number of selected ranges, emptiness, or time).
function addSelectionToHistory(doc, sel, opId, options) {
  let hist = doc.history, origin = options && options.origin

  // A new event is started when the previous origin does not match
  // the current, or the origins don't allow matching. Origins
  // starting with * are always merged, those starting with + are
  // merged when similar and close together in time.
  if (opId == hist.lastSelOp ||
      (origin && hist.lastSelOrigin == origin &&
       (hist.lastModTime == hist.lastSelTime && hist.lastOrigin == origin ||
        selectionEventCanBeMerged(doc, origin, (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(hist.done), sel))))
    hist.done[hist.done.length - 1] = sel
  else
    pushSelectionToHistory(sel, hist.done)

  hist.lastSelTime = +new Date
  hist.lastSelOrigin = origin
  hist.lastSelOp = opId
  if (options && options.clearRedo !== false)
    clearSelectionEvents(hist.undone)
}

function pushSelectionToHistory(sel, dest) {
  let top = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(dest)
  if (!(top && top.ranges && top.equals(sel)))
    dest.push(sel)
}

// Used to store marked span information in the history.
function attachLocalSpans(doc, change, from, to) {
  let existing = change["spans_" + doc.id], n = 0
  doc.iter(Math.max(doc.first, from), Math.min(doc.first + doc.size, to), line => {
    if (line.markedSpans)
      (existing || (existing = change["spans_" + doc.id] = {}))[n] = line.markedSpans
    ++n
  })
}

// When un/re-doing restores text containing marked spans, those
// that have been explicitly cleared should not be restored.
function removeClearedSpans(spans) {
  if (!spans) return null
  let out
  for (let i = 0; i < spans.length; ++i) {
    if (spans[i].marker.explicitlyCleared) { if (!out) out = spans.slice(0, i) }
    else if (out) out.push(spans[i])
  }
  return !out ? spans : out.length ? out : null
}

// Retrieve and filter the old marked spans stored in a change event.
function getOldSpans(doc, change) {
  let found = change["spans_" + doc.id]
  if (!found) return null
  let nw = []
  for (let i = 0; i < change.text.length; ++i)
    nw.push(removeClearedSpans(found[i]))
  return nw
}

// Used for un/re-doing changes from the history. Combines the
// result of computing the existing spans with the set of spans that
// existed in the history (so that deleting around a span and then
// undoing brings back the span).
function mergeOldSpans(doc, change) {
  let old = getOldSpans(doc, change)
  let stretched = (0,_line_spans_js__WEBPACK_IMPORTED_MODULE_1__.stretchSpansOverChange)(doc, change)
  if (!old) return stretched
  if (!stretched) return old

  for (let i = 0; i < old.length; ++i) {
    let oldCur = old[i], stretchCur = stretched[i]
    if (oldCur && stretchCur) {
      spans: for (let j = 0; j < stretchCur.length; ++j) {
        let span = stretchCur[j]
        for (let k = 0; k < oldCur.length; ++k)
          if (oldCur[k].marker == span.marker) continue spans
        oldCur.push(span)
      }
    } else if (stretchCur) {
      old[i] = stretchCur
    }
  }
  return old
}

// Used both to provide a JSON-safe object in .getHistory, and, when
// detaching a document, to split the history in two
function copyHistoryArray(events, newGroup, instantiateSel) {
  let copy = []
  for (let i = 0; i < events.length; ++i) {
    let event = events[i]
    if (event.ranges) {
      copy.push(instantiateSel ? _selection_js__WEBPACK_IMPORTED_MODULE_7__.Selection.prototype.deepCopy.call(event) : event)
      continue
    }
    let changes = event.changes, newChanges = []
    copy.push({changes: newChanges})
    for (let j = 0; j < changes.length; ++j) {
      let change = changes[j], m
      newChanges.push({from: change.from, to: change.to, text: change.text})
      if (newGroup) for (var prop in change) if (m = prop.match(/^spans_(\d+)$/)) {
        if ((0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.indexOf)(newGroup, Number(m[1])) > -1) {
          (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_4__.lst)(newChanges)[prop] = change[prop]
          delete change[prop]
        }
      }
    }
  }
  return copy
}


/***/ }),

/***/ "./node_modules/codemirror/src/model/selection.js":
/*!********************************************************!*\
  !*** ./node_modules/codemirror/src/model/selection.js ***!
  \********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "Selection": () => (/* binding */ Selection),
/* harmony export */   "Range": () => (/* binding */ Range),
/* harmony export */   "normalizeSelection": () => (/* binding */ normalizeSelection),
/* harmony export */   "simpleSelection": () => (/* binding */ simpleSelection)
/* harmony export */ });
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");



// Selection objects are immutable. A new one is created every time
// the selection changes. A selection is one or more non-overlapping
// (and non-touching) ranges, sorted, and an integer that indicates
// which one is the primary selection (the one that's scrolled into
// view, that getCursor returns, etc).
class Selection {
  constructor(ranges, primIndex) {
    this.ranges = ranges
    this.primIndex = primIndex
  }

  primary() { return this.ranges[this.primIndex] }

  equals(other) {
    if (other == this) return true
    if (other.primIndex != this.primIndex || other.ranges.length != this.ranges.length) return false
    for (let i = 0; i < this.ranges.length; i++) {
      let here = this.ranges[i], there = other.ranges[i]
      if (!(0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.equalCursorPos)(here.anchor, there.anchor) || !(0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.equalCursorPos)(here.head, there.head)) return false
    }
    return true
  }

  deepCopy() {
    let out = []
    for (let i = 0; i < this.ranges.length; i++)
      out[i] = new Range((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.copyPos)(this.ranges[i].anchor), (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.copyPos)(this.ranges[i].head))
    return new Selection(out, this.primIndex)
  }

  somethingSelected() {
    for (let i = 0; i < this.ranges.length; i++)
      if (!this.ranges[i].empty()) return true
    return false
  }

  contains(pos, end) {
    if (!end) end = pos
    for (let i = 0; i < this.ranges.length; i++) {
      let range = this.ranges[i]
      if ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.cmp)(end, range.from()) >= 0 && (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.cmp)(pos, range.to()) <= 0)
        return i
    }
    return -1
  }
}

class Range {
  constructor(anchor, head) {
    this.anchor = anchor; this.head = head
  }

  from() { return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.minPos)(this.anchor, this.head) }
  to() { return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.maxPos)(this.anchor, this.head) }
  empty() { return this.head.line == this.anchor.line && this.head.ch == this.anchor.ch }
}

// Take an unsorted, potentially overlapping set of ranges, and
// build a selection out of it. 'Consumes' ranges array (modifying
// it).
function normalizeSelection(cm, ranges, primIndex) {
  let mayTouch = cm && cm.options.selectionsMayTouch
  let prim = ranges[primIndex]
  ranges.sort((a, b) => (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.cmp)(a.from(), b.from()))
  primIndex = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_1__.indexOf)(ranges, prim)
  for (let i = 1; i < ranges.length; i++) {
    let cur = ranges[i], prev = ranges[i - 1]
    let diff = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.cmp)(prev.to(), cur.from())
    if (mayTouch && !cur.empty() ? diff > 0 : diff >= 0) {
      let from = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.minPos)(prev.from(), cur.from()), to = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_0__.maxPos)(prev.to(), cur.to())
      let inv = prev.empty() ? cur.from() == cur.head : prev.from() == prev.head
      if (i <= primIndex) --primIndex
      ranges.splice(--i, 2, new Range(inv ? to : from, inv ? from : to))
    }
  }
  return new Selection(ranges, primIndex)
}

function simpleSelection(anchor, head) {
  return new Selection([new Range(anchor, head || anchor)], 0)
}


/***/ }),

/***/ "./node_modules/codemirror/src/model/selection_updates.js":
/*!****************************************************************!*\
  !*** ./node_modules/codemirror/src/model/selection_updates.js ***!
  \****************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "extendRange": () => (/* binding */ extendRange),
/* harmony export */   "extendSelection": () => (/* binding */ extendSelection),
/* harmony export */   "extendSelections": () => (/* binding */ extendSelections),
/* harmony export */   "replaceOneSelection": () => (/* binding */ replaceOneSelection),
/* harmony export */   "setSimpleSelection": () => (/* binding */ setSimpleSelection),
/* harmony export */   "setSelectionReplaceHistory": () => (/* binding */ setSelectionReplaceHistory),
/* harmony export */   "setSelection": () => (/* binding */ setSelection),
/* harmony export */   "setSelectionNoUndo": () => (/* binding */ setSelectionNoUndo),
/* harmony export */   "reCheckSelection": () => (/* binding */ reCheckSelection),
/* harmony export */   "skipAtomic": () => (/* binding */ skipAtomic),
/* harmony export */   "selectAll": () => (/* binding */ selectAll)
/* harmony export */ });
/* harmony import */ var _util_operation_group_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ../util/operation_group.js */ "./node_modules/codemirror/src/util/operation_group.js");
/* harmony import */ var _display_scrolling_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ../display/scrolling.js */ "./node_modules/codemirror/src/display/scrolling.js");
/* harmony import */ var _line_pos_js__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../line/pos.js */ "./node_modules/codemirror/src/line/pos.js");
/* harmony import */ var _line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../line/utils_line.js */ "./node_modules/codemirror/src/line/utils_line.js");
/* harmony import */ var _util_event_js__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! ../util/event.js */ "./node_modules/codemirror/src/util/event.js");
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! ../util/misc.js */ "./node_modules/codemirror/src/util/misc.js");
/* harmony import */ var _history_js__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! ./history.js */ "./node_modules/codemirror/src/model/history.js");
/* harmony import */ var _selection_js__WEBPACK_IMPORTED_MODULE_7__ = __webpack_require__(/*! ./selection.js */ "./node_modules/codemirror/src/model/selection.js");










// The 'scroll' parameter given to many of these indicated whether
// the new cursor position should be scrolled into view after
// modifying the selection.

// If shift is held or the extend flag is set, extends a range to
// include a given position (and optionally a second position).
// Otherwise, simply returns the range between the given positions.
// Used for cursor motion and such.
function extendRange(range, head, other, extend) {
  if (extend) {
    let anchor = range.anchor
    if (other) {
      let posBefore = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.cmp)(head, anchor) < 0
      if (posBefore != ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.cmp)(other, anchor) < 0)) {
        anchor = head
        head = other
      } else if (posBefore != ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.cmp)(head, other) < 0)) {
        head = other
      }
    }
    return new _selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(anchor, head)
  } else {
    return new _selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(other || head, head)
  }
}

// Extend the primary selection range, discard the rest.
function extendSelection(doc, head, other, options, extend) {
  if (extend == null) extend = doc.cm && (doc.cm.display.shift || doc.extend)
  setSelection(doc, new _selection_js__WEBPACK_IMPORTED_MODULE_7__.Selection([extendRange(doc.sel.primary(), head, other, extend)], 0), options)
}

// Extend all selections (pos is an array of selections with length
// equal the number of selections)
function extendSelections(doc, heads, options) {
  let out = []
  let extend = doc.cm && (doc.cm.display.shift || doc.extend)
  for (let i = 0; i < doc.sel.ranges.length; i++)
    out[i] = extendRange(doc.sel.ranges[i], heads[i], null, extend)
  let newSel = (0,_selection_js__WEBPACK_IMPORTED_MODULE_7__.normalizeSelection)(doc.cm, out, doc.sel.primIndex)
  setSelection(doc, newSel, options)
}

// Updates a single range in the selection.
function replaceOneSelection(doc, i, range, options) {
  let ranges = doc.sel.ranges.slice(0)
  ranges[i] = range
  setSelection(doc, (0,_selection_js__WEBPACK_IMPORTED_MODULE_7__.normalizeSelection)(doc.cm, ranges, doc.sel.primIndex), options)
}

// Reset the selection to a single range.
function setSimpleSelection(doc, anchor, head, options) {
  setSelection(doc, (0,_selection_js__WEBPACK_IMPORTED_MODULE_7__.simpleSelection)(anchor, head), options)
}

// Give beforeSelectionChange handlers a change to influence a
// selection update.
function filterSelectionChange(doc, sel, options) {
  let obj = {
    ranges: sel.ranges,
    update: function(ranges) {
      this.ranges = []
      for (let i = 0; i < ranges.length; i++)
        this.ranges[i] = new _selection_js__WEBPACK_IMPORTED_MODULE_7__.Range((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.clipPos)(doc, ranges[i].anchor),
                                   (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.clipPos)(doc, ranges[i].head))
    },
    origin: options && options.origin
  }
  ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.signal)(doc, "beforeSelectionChange", doc, obj)
  if (doc.cm) (0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.signal)(doc.cm, "beforeSelectionChange", doc.cm, obj)
  if (obj.ranges != sel.ranges) return (0,_selection_js__WEBPACK_IMPORTED_MODULE_7__.normalizeSelection)(doc.cm, obj.ranges, obj.ranges.length - 1)
  else return sel
}

function setSelectionReplaceHistory(doc, sel, options) {
  let done = doc.history.done, last = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_5__.lst)(done)
  if (last && last.ranges) {
    done[done.length - 1] = sel
    setSelectionNoUndo(doc, sel, options)
  } else {
    setSelection(doc, sel, options)
  }
}

// Set a new selection.
function setSelection(doc, sel, options) {
  setSelectionNoUndo(doc, sel, options)
  ;(0,_history_js__WEBPACK_IMPORTED_MODULE_6__.addSelectionToHistory)(doc, doc.sel, doc.cm ? doc.cm.curOp.id : NaN, options)
}

function setSelectionNoUndo(doc, sel, options) {
  if ((0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.hasHandler)(doc, "beforeSelectionChange") || doc.cm && (0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.hasHandler)(doc.cm, "beforeSelectionChange"))
    sel = filterSelectionChange(doc, sel, options)

  let bias = options && options.bias ||
    ((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.cmp)(sel.primary().head, doc.sel.primary().head) < 0 ? -1 : 1)
  setSelectionInner(doc, skipAtomicInSelection(doc, sel, bias, true))

  if (!(options && options.scroll === false) && doc.cm && doc.cm.getOption("readOnly") != "nocursor")
    (0,_display_scrolling_js__WEBPACK_IMPORTED_MODULE_1__.ensureCursorVisible)(doc.cm)
}

function setSelectionInner(doc, sel) {
  if (sel.equals(doc.sel)) return

  doc.sel = sel

  if (doc.cm) {
    doc.cm.curOp.updateInput = 1
    doc.cm.curOp.selectionChanged = true
    ;(0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.signalCursorActivity)(doc.cm)
  }
  (0,_util_operation_group_js__WEBPACK_IMPORTED_MODULE_0__.signalLater)(doc, "cursorActivity", doc)
}

// Verify that the selection does not partially select any atomic
// marked ranges.
function reCheckSelection(doc) {
  setSelectionInner(doc, skipAtomicInSelection(doc, doc.sel, null, false))
}

// Return a selection that does not partially select any atomic
// ranges.
function skipAtomicInSelection(doc, sel, bias, mayClear) {
  let out
  for (let i = 0; i < sel.ranges.length; i++) {
    let range = sel.ranges[i]
    let old = sel.ranges.length == doc.sel.ranges.length && doc.sel.ranges[i]
    let newAnchor = skipAtomic(doc, range.anchor, old && old.anchor, bias, mayClear)
    let newHead = skipAtomic(doc, range.head, old && old.head, bias, mayClear)
    if (out || newAnchor != range.anchor || newHead != range.head) {
      if (!out) out = sel.ranges.slice(0, i)
      out[i] = new _selection_js__WEBPACK_IMPORTED_MODULE_7__.Range(newAnchor, newHead)
    }
  }
  return out ? (0,_selection_js__WEBPACK_IMPORTED_MODULE_7__.normalizeSelection)(doc.cm, out, sel.primIndex) : sel
}

function skipAtomicInner(doc, pos, oldPos, dir, mayClear) {
  let line = (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, pos.line)
  if (line.markedSpans) for (let i = 0; i < line.markedSpans.length; ++i) {
    let sp = line.markedSpans[i], m = sp.marker

    // Determine if we should prevent the cursor being placed to the left/right of an atomic marker
    // Historically this was determined using the inclusiveLeft/Right option, but the new way to control it
    // is with selectLeft/Right
    let preventCursorLeft = ("selectLeft" in m) ? !m.selectLeft : m.inclusiveLeft
    let preventCursorRight = ("selectRight" in m) ? !m.selectRight : m.inclusiveRight

    if ((sp.from == null || (preventCursorLeft ? sp.from <= pos.ch : sp.from < pos.ch)) &&
        (sp.to == null || (preventCursorRight ? sp.to >= pos.ch : sp.to > pos.ch))) {
      if (mayClear) {
        (0,_util_event_js__WEBPACK_IMPORTED_MODULE_4__.signal)(m, "beforeCursorEnter")
        if (m.explicitlyCleared) {
          if (!line.markedSpans) break
          else {--i; continue}
        }
      }
      if (!m.atomic) continue

      if (oldPos) {
        let near = m.find(dir < 0 ? 1 : -1), diff
        if (dir < 0 ? preventCursorRight : preventCursorLeft)
          near = movePos(doc, near, -dir, near && near.line == pos.line ? line : null)
        if (near && near.line == pos.line && (diff = (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.cmp)(near, oldPos)) && (dir < 0 ? diff < 0 : diff > 0))
          return skipAtomicInner(doc, near, pos, dir, mayClear)
      }

      let far = m.find(dir < 0 ? -1 : 1)
      if (dir < 0 ? preventCursorLeft : preventCursorRight)
        far = movePos(doc, far, dir, far.line == pos.line ? line : null)
      return far ? skipAtomicInner(doc, far, pos, dir, mayClear) : null
    }
  }
  return pos
}

// Ensure a given position is not inside an atomic range.
function skipAtomic(doc, pos, oldPos, bias, mayClear) {
  let dir = bias || 1
  let found = skipAtomicInner(doc, pos, oldPos, dir, mayClear) ||
      (!mayClear && skipAtomicInner(doc, pos, oldPos, dir, true)) ||
      skipAtomicInner(doc, pos, oldPos, -dir, mayClear) ||
      (!mayClear && skipAtomicInner(doc, pos, oldPos, -dir, true))
  if (!found) {
    doc.cantEdit = true
    return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(doc.first, 0)
  }
  return found
}

function movePos(doc, pos, dir, line) {
  if (dir < 0 && pos.ch == 0) {
    if (pos.line > doc.first) return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.clipPos)(doc, (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(pos.line - 1))
    else return null
  } else if (dir > 0 && pos.ch == (line || (0,_line_utils_line_js__WEBPACK_IMPORTED_MODULE_3__.getLine)(doc, pos.line)).text.length) {
    if (pos.line < doc.first + doc.size - 1) return (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(pos.line + 1, 0)
    else return null
  } else {
    return new _line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos(pos.line, pos.ch + dir)
  }
}

function selectAll(cm) {
  cm.setSelection((0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(cm.firstLine(), 0), (0,_line_pos_js__WEBPACK_IMPORTED_MODULE_2__.Pos)(cm.lastLine()), _util_misc_js__WEBPACK_IMPORTED_MODULE_5__.sel_dontScroll)
}


/***/ }),

/***/ "./node_modules/codemirror/src/modes.js":
/*!**********************************************!*\
  !*** ./node_modules/codemirror/src/modes.js ***!
  \**********************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "modes": () => (/* binding */ modes),
/* harmony export */   "mimeModes": () => (/* binding */ mimeModes),
/* harmony export */   "defineMode": () => (/* binding */ defineMode),
/* harmony export */   "defineMIME": () => (/* binding */ defineMIME),
/* harmony export */   "resolveMode": () => (/* binding */ resolveMode),
/* harmony export */   "getMode": () => (/* binding */ getMode),
/* harmony export */   "modeExtensions": () => (/* binding */ modeExtensions),
/* harmony export */   "extendMode": () => (/* binding */ extendMode),
/* harmony export */   "copyState": () => (/* binding */ copyState),
/* harmony export */   "innerMode": () => (/* binding */ innerMode),
/* harmony export */   "startState": () => (/* binding */ startState)
/* harmony export */ });
/* harmony import */ var _util_misc_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./util/misc.js */ "./node_modules/codemirror/src/util/misc.js");


// Known modes, by name and by MIME
let modes = {}, mimeModes = {}

// Extra arguments are stored as the mode's dependencies, which is
// used by (legacy) mechanisms like loadmode.js to automatically
// load a mode. (Preferred mechanism is the require/define calls.)
function defineMode(name, mode) {
  if (arguments.length > 2)
    mode.dependencies = Array.prototype.slice.call(arguments, 2)
  modes[name] = mode
}

function defineMIME(mime, spec) {
  mimeModes[mime] = spec
}

// Given a MIME type, a {name, ...options} config object, or a name
// string, return a mode config object.
function resolveMode(spec) {
  if (typeof spec == "string" && mimeModes.hasOwnProperty(spec)) {
    spec = mimeModes[spec]
  } else if (spec && typeof spec.name == "string" && mimeModes.hasOwnProperty(spec.name)) {
    let found = mimeModes[spec.name]
    if (typeof found == "string") found = {name: found}
    spec = (0,_util_misc_js__WEBPACK_IMPORTED_MODULE_0__.createObj)(found, spec)
    spec.name = found.name
  } else if (typeof spec == "string" && /^[\w\-]+\/[\w\-]+\+xml$/.test(spec)) {
    return resolveMode("application/xml")
  } else if (typeof spec == "string" && /^[\w\-]+\/[\w\-]+\+json$/.test(spec)) {
    return resolveMode("application/json")
  }
  if (typeof spec == "string") return {name: spec}
  else return spec || {name: "null"}
}

// Given a mode spec (anything that resolveMode accepts), find and
// initialize an actual mode object.
function getMode(options, spec) {
  spec = resolveMode(spec)
  let mfactory = modes[spec.name]
  if (!mfactory) return getMode(options, "text/plain")
  let modeObj = mfactory(options, spec)
  if (modeExtensions.hasOwnProperty(spec.name)) {
    let exts = modeExtensions[spec.name]
    for (let prop in exts) {
      if (!exts.hasOwnProperty(prop)) continue
      if (modeObj.hasOwnProperty(prop)) modeObj["_" + prop] = modeObj[prop]
      modeObj[prop] = exts[prop]
    }
  }
  modeObj.name = spec.name
  if (spec.helperType) modeObj.helperType = spec.helperType
  if (spec.modeProps) for (let prop in spec.modeProps)
    modeObj[prop] = spec.modeProps[prop]

  return modeObj
}

// This can be used to attach properties to mode objects from
// outside the actual mode definition.
let modeExtensions = {}
function extendMode(mode, properties) {
  let exts = modeExtensions.hasOwnProperty(mode) ? modeExtensions[mode] : (modeExtensions[mode] = {})
  ;(0,_util_misc_js__WEBPACK_IMPORTED_MODULE_0__.copyObj)(properties, exts)
}

function copyState(mode, state) {
  if (state === true) return state
  if (mode.copyState) return mode.copyState(state)
  let nstate = {}
  for (let n in state) {
    let val = state[n]
    if (val instanceof Array) val = val.concat([])
    nstate[n] = val
  }
  return nstate
}

// Given a mode and a state (for that mode), find the inner mode and
// state at the position that the state refers to.
function innerMode(mode, state) {
  let info
  while (mode.innerMode) {
    info = mode.innerMode(state)
    if (!info || info.mode == mode) break
    state = info.state
    mode = info.mode
  }
  return info || {mode: mode, state: state}
}

function startState(mode, a1, a2) {
  return mode.startState ? mode.startState(a1, a2) : true
}


/***/ }),

/***/ "./node_modules/codemirror/src/util/StringStream.js":
/*!**********************************************************!*\
  !*** ./node_modules/codemirror/src/util/StringStream.js ***!
  \**********************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "default": () => (__WEBPACK_DEFAULT_EXPORT__)
/* harmony export */ });
/* harmony import */ var _misc_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./misc.js */ "./node_modules/codemirror/src/util/misc.js");


// STRING STREAM

// Fed to the mode parsers, provides helper functions to make
// parsers more succinct.

class StringStream {
  constructor(string, tabSize, lineOracle) {
    this.pos = this.start = 0
    this.string = string
    this.tabSize = tabSize || 8
    this.lastColumnPos = this.lastColumnValue = 0
    this.lineStart = 0
    this.lineOracle = lineOracle
  }

  eol() {return this.pos >= this.string.length}
  sol() {return this.pos == this.lineStart}
  peek() {return this.string.charAt(this.pos) || undefined}
  next() {
    if (this.pos < this.string.length)
      return this.string.charAt(this.pos++)
  }
  eat(match) {
    let ch = this.string.charAt(this.pos)
    let ok
    if (typeof match == "string") ok = ch == match
    else ok = ch && (match.test ? match.test(ch) : match(ch))
    if (ok) {++this.pos; return ch}
  }
  eatWhile(match) {
    let start = this.pos
    while (this.eat(match)){}
    return this.pos > start
  }
  eatSpace() {
    let start = this.pos
    while (/[\s\u00a0]/.test(this.string.charAt(this.pos))) ++this.pos
    return this.pos > start
  }
  skipToEnd() {this.pos = this.string.length}
  skipTo(ch) {
    let found = this.string.indexOf(ch, this.pos)
    if (found > -1) {this.pos = found; return true}
  }
  backUp(n) {this.pos -= n}
  column() {
    if (this.lastColumnPos < this.start) {
      this.lastColumnValue = (0,_misc_js__WEBPACK_IMPORTED_MODULE_0__.countColumn)(this.string, this.start, this.tabSize, this.lastColumnPos, this.lastColumnValue)
      this.lastColumnPos = this.start
    }
    return this.lastColumnValue - (this.lineStart ? (0,_misc_js__WEBPACK_IMPORTED_MODULE_0__.countColumn)(this.string, this.lineStart, this.tabSize) : 0)
  }
  indentation() {
    return (0,_misc_js__WEBPACK_IMPORTED_MODULE_0__.countColumn)(this.string, null, this.tabSize) -
      (this.lineStart ? (0,_misc_js__WEBPACK_IMPORTED_MODULE_0__.countColumn)(this.string, this.lineStart, this.tabSize) : 0)
  }
  match(pattern, consume, caseInsensitive) {
    if (typeof pattern == "string") {
      let cased = str => caseInsensitive ? str.toLowerCase() : str
      let substr = this.string.substr(this.pos, pattern.length)
      if (cased(substr) == cased(pattern)) {
        if (consume !== false) this.pos += pattern.length
        return true
      }
    } else {
      let match = this.string.slice(this.pos).match(pattern)
      if (match && match.index > 0) return null
      if (match && consume !== false) this.pos += match[0].length
      return match
    }
  }
  current(){return this.string.slice(this.start, this.pos)}
  hideFirstChars(n, inner) {
    this.lineStart += n
    try { return inner() }
    finally { this.lineStart -= n }
  }
  lookAhead(n) {
    let oracle = this.lineOracle
    return oracle && oracle.lookAhead(n)
  }
  baseToken() {
    let oracle = this.lineOracle
    return oracle && oracle.baseToken(this.pos)
  }
}

/* harmony default export */ const __WEBPACK_DEFAULT_EXPORT__ = (StringStream);


/***/ }),

/***/ "./node_modules/codemirror/src/util/bidi.js":
/*!**************************************************!*\
  !*** ./node_modules/codemirror/src/util/bidi.js ***!
  \**************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "iterateBidiSections": () => (/* binding */ iterateBidiSections),
/* harmony export */   "bidiOther": () => (/* binding */ bidiOther),
/* harmony export */   "getBidiPartAt": () => (/* binding */ getBidiPartAt),
/* harmony export */   "getOrder": () => (/* binding */ getOrder)
/* harmony export */ });
/* harmony import */ var _misc_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./misc.js */ "./node_modules/codemirror/src/util/misc.js");


// BIDI HELPERS

function iterateBidiSections(order, from, to, f) {
  if (!order) return f(from, to, "ltr", 0)
  let found = false
  for (let i = 0; i < order.length; ++i) {
    let part = order[i]
    if (part.from < to && part.to > from || from == to && part.to == from) {
      f(Math.max(part.from, from), Math.min(part.to, to), part.level == 1 ? "rtl" : "ltr", i)
      found = true
    }
  }
  if (!found) f(from, to, "ltr")
}

let bidiOther = null
function getBidiPartAt(order, ch, sticky) {
  let found
  bidiOther = null
  for (let i = 0; i < order.length; ++i) {
    let cur = order[i]
    if (cur.from < ch && cur.to > ch) return i
    if (cur.to == ch) {
      if (cur.from != cur.to && sticky == "before") found = i
      else bidiOther = i
    }
    if (cur.from == ch) {
      if (cur.from != cur.to && sticky != "before") found = i
      else bidiOther = i
    }
  }
  return found != null ? found : bidiOther
}

// Bidirectional ordering algorithm
// See http://unicode.org/reports/tr9/tr9-13.html for the algorithm
// that this (partially) implements.

// One-char codes used for character types:
// L (L):   Left-to-Right
// R (R):   Right-to-Left
// r (AL):  Right-to-Left Arabic
// 1 (EN):  European Number
// + (ES):  European Number Separator
// % (ET):  European Number Terminator
// n (AN):  Arabic Number
// , (CS):  Common Number Separator
// m (NSM): Non-Spacing Mark
// b (BN):  Boundary Neutral
// s (B):   Paragraph Separator
// t (S):   Segment Separator
// w (WS):  Whitespace
// N (ON):  Other Neutrals

// Returns null if characters are ordered as they appear
// (left-to-right), or an array of sections ({from, to, level}
// objects) in the order in which they occur visually.
let bidiOrdering = (function() {
  // Character types for codepoints 0 to 0xff
  let lowTypes = "bbbbbbbbbtstwsbbbbbbbbbbbbbbssstwNN%%%NNNNNN,N,N1111111111NNNNNNNLLLLLLLLLLLLLLLLLLLLLLLLLLNNNNNNLLLLLLLLLLLLLLLLLLLLLLLLLLNNNNbbbbbbsbbbbbbbbbbbbbbbbbbbbbbbbbb,N%%%%NNNNLNNNNN%%11NLNNN1LNNNNNLLLLLLLLLLLLLLLLLLLLLLLNLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLN"
  // Character types for codepoints 0x600 to 0x6f9
  let arabicTypes = "nnnnnnNNr%%r,rNNmmmmmmmmmmmrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrmmmmmmmmmmmmmmmmmmmmmnnnnnnnnnn%nnrrrmrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrmmmmmmmnNmmmmmmrrmmNmmmmrr1111111111"
  function charType(code) {
    if (code <= 0xf7) return lowTypes.charAt(code)
    else if (0x590 <= code && code <= 0x5f4) return "R"
    else if (0x600 <= code && code <= 0x6f9) return arabicTypes.charAt(code - 0x600)
    else if (0x6ee <= code && code <= 0x8ac) return "r"
    else if (0x2000 <= code && code <= 0x200b) return "w"
    else if (code == 0x200c) return "b"
    else return "L"
  }

  let bidiRE = /[\u0590-\u05f4\u0600-\u06ff\u0700-\u08ac]/
  let isNeutral = /[stwN]/, isStrong = /[LRr]/, countsAsLeft = /[Lb1n]/, countsAsNum = /[1n]/

  function BidiSpan(level, from, to) {
    this.level = level
    this.from = from; this.to = to
  }

  return function(str, direction) {
    let outerType = direction == "ltr" ? "L" : "R"

    if (str.length == 0 || direction == "ltr" && !bidiRE.test(str)) return false
    let len = str.length, types = []
    for (let i = 0; i < len; ++i)
      types.push(charType(str.charCodeAt(i)))

    // W1. Examine each non-spacing mark (NSM) in the level run, and
    // change the type of the NSM to the type of the previous
    // character. If the NSM is at the start of the level run, it will
    // get the type of sor.
    for (let i = 0, prev = outerType; i < len; ++i) {
      let type = types[i]
      if (type == "m") types[i] = prev
      else prev = type
    }

    // W2. Search backwards from each instance of a European number
    // until the first strong type (R, L, AL, or sor) is found. If an
    // AL is found, change the type of the European number to Arabic
    // number.
    // W3. Change all ALs to R.
    for (let i = 0, cur = outerType; i < len; ++i) {
      let type = types[i]
      if (type == "1" && cur == "r") types[i] = "n"
      else if (isStrong.test(type)) { cur = type; if (type == "r") types[i] = "R" }
    }

    // W4. A single European separator between two European numbers
    // changes to a European number. A single common separator between
    // two numbers of the same type changes to that type.
    for (let i = 1, prev = types[0]; i < len - 1; ++i) {
      let type = types[i]
      if (type == "+" && prev == "1" && types[i+1] == "1") types[i] = "1"
      else if (type == "," && prev == types[i+1] &&
               (prev == "1" || prev == "n")) types[i] = prev
      prev = type
    }

    // W5. A sequence of European terminators adjacent to European
    // numbers changes to all European numbers.
    // W6. Otherwise, separators and terminators change to Other
    // Neutral.
    for (let i = 0; i < len; ++i) {
      let type = types[i]
      if (type == ",") types[i] = "N"
      else if (type == "%") {
        let end
        for (end = i + 1; end < len && types[end] == "%"; ++end) {}
        let replace = (i && types[i-1] == "!") || (end < len && types[end] == "1") ? "1" : "N"
        for (let j = i; j < end; ++j) types[j] = replace
        i = end - 1
      }
    }

    // W7. Search backwards from each instance of a European number
    // until the first strong type (R, L, or sor) is found. If an L is
    // found, then change the type of the European number to L.
    for (let i = 0, cur = outerType; i < len; ++i) {
      let type = types[i]
      if (cur == "L" && type == "1") types[i] = "L"
      else if (isStrong.test(type)) cur = type
    }

    // N1. A sequence of neutrals takes the direction of the
    // surrounding strong text if the text on both sides has the same
    // direction. European and Arabic numbers act as if they were R in
    // terms of their influence on neutrals. Start-of-level-run (sor)
    // and end-of-level-run (eor) are used at level run boundaries.
    // N2. Any remaining neutrals take the embedding direction.
    for (let i = 0; i < len; ++i) {
      if (isNeutral.test(types[i])) {
        let end
        for (end = i + 1; end < len && isNeutral.test(types[end]); ++end) {}
        let before = (i ? types[i-1] : outerType) == "L"
        let after = (end < len ? types[end] : outerType) == "L"
        let replace = before == after ? (before ? "L" : "R") : outerType
        for (let j = i; j < end; ++j) types[j] = replace
        i = end - 1
      }
    }

    // Here we depart from the documented algorithm, in order to avoid
    // building up an actual levels array. Since there are only three
    // levels (0, 1, 2) in an implementation that doesn't take
    // explicit embedding into account, we can build up the order on
    // the fly, without following the level-based algorithm.
    let order = [], m
    for (let i = 0; i < len;) {
      if (countsAsLeft.test(types[i])) {
        let start = i
        for (++i; i < len && countsAsLeft.test(types[i]); ++i) {}
        order.push(new BidiSpan(0, start, i))
      } else {
        let pos = i, at = order.length, isRTL = direction == "rtl" ? 1 : 0
        for (++i; i < len && types[i] != "L"; ++i) {}
        for (let j = pos; j < i;) {
          if (countsAsNum.test(types[j])) {
            if (pos < j) { order.splice(at, 0, new BidiSpan(1, pos, j)); at += isRTL }
            let nstart = j
            for (++j; j < i && countsAsNum.test(types[j]); ++j) {}
            order.splice(at, 0, new BidiSpan(2, nstart, j))
            at += isRTL
            pos = j
          } else ++j
        }
        if (pos < i) order.splice(at, 0, new BidiSpan(1, pos, i))
      }
    }
    if (direction == "ltr") {
      if (order[0].level == 1 && (m = str.match(/^\s+/))) {
        order[0].from = m[0].length
        order.unshift(new BidiSpan(0, 0, m[0].length))
      }
      if ((0,_misc_js__WEBPACK_IMPORTED_MODULE_0__.lst)(order).level == 1 && (m = str.match(/\s+$/))) {
        (0,_misc_js__WEBPACK_IMPORTED_MODULE_0__.lst)(order).to -= m[0].length
        order.push(new BidiSpan(0, len - m[0].length, len))
      }
    }

    return direction == "rtl" ? order.reverse() : order
  }
})()

// Get the bidi ordering for the given line (and cache it). Returns
// false for lines that are fully left-to-right, and an array of
// BidiSpan objects otherwise.
function getOrder(line, direction) {
  let order = line.order
  if (order == null) order = line.order = bidiOrdering(line.text, direction)
  return order
}


/***/ }),

/***/ "./node_modules/codemirror/src/util/browser.js":
/*!*****************************************************!*\
  !*** ./node_modules/codemirror/src/util/browser.js ***!
  \*****************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "gecko": () => (/* binding */ gecko),
/* harmony export */   "ie": () => (/* binding */ ie),
/* harmony export */   "ie_version": () => (/* binding */ ie_version),
/* harmony export */   "webkit": () => (/* binding */ webkit),
/* harmony export */   "chrome": () => (/* binding */ chrome),
/* harmony export */   "presto": () => (/* binding */ presto),
/* harmony export */   "safari": () => (/* binding */ safari),
/* harmony export */   "mac_geMountainLion": () => (/* binding */ mac_geMountainLion),
/* harmony export */   "phantom": () => (/* binding */ phantom),
/* harmony export */   "ios": () => (/* binding */ ios),
/* harmony export */   "android": () => (/* binding */ android),
/* harmony export */   "mobile": () => (/* binding */ mobile),
/* harmony export */   "mac": () => (/* binding */ mac),
/* harmony export */   "chromeOS": () => (/* binding */ chromeOS),
/* harmony export */   "windows": () => (/* binding */ windows),
/* harmony export */   "flipCtrlCmd": () => (/* binding */ flipCtrlCmd),
/* harmony export */   "captureRightClick": () => (/* binding */ captureRightClick)
/* harmony export */ });
// Kludges for bugs and behavior differences that can't be feature
// detected are enabled based on userAgent etc sniffing.
let userAgent = navigator.userAgent
let platform = navigator.platform

let gecko = /gecko\/\d/i.test(userAgent)
let ie_upto10 = /MSIE \d/.test(userAgent)
let ie_11up = /Trident\/(?:[7-9]|\d{2,})\..*rv:(\d+)/.exec(userAgent)
let edge = /Edge\/(\d+)/.exec(userAgent)
let ie = ie_upto10 || ie_11up || edge
let ie_version = ie && (ie_upto10 ? document.documentMode || 6 : +(edge || ie_11up)[1])
let webkit = !edge && /WebKit\//.test(userAgent)
let qtwebkit = webkit && /Qt\/\d+\.\d+/.test(userAgent)
let chrome = !edge && /Chrome\//.test(userAgent)
let presto = /Opera\//.test(userAgent)
let safari = /Apple Computer/.test(navigator.vendor)
let mac_geMountainLion = /Mac OS X 1\d\D([8-9]|\d\d)\D/.test(userAgent)
let phantom = /PhantomJS/.test(userAgent)

let ios = safari && (/Mobile\/\w+/.test(userAgent) || navigator.maxTouchPoints > 2)
let android = /Android/.test(userAgent)
// This is woefully incomplete. Suggestions for alternative methods welcome.
let mobile = ios || android || /webOS|BlackBerry|Opera Mini|Opera Mobi|IEMobile/i.test(userAgent)
let mac = ios || /Mac/.test(platform)
let chromeOS = /\bCrOS\b/.test(userAgent)
let windows = /win/i.test(platform)

let presto_version = presto && userAgent.match(/Version\/(\d*\.\d*)/)
if (presto_version) presto_version = Number(presto_version[1])
if (presto_version && presto_version >= 15) { presto = false; webkit = true }
// Some browsers use the wrong event properties to signal cmd/ctrl on OS X
let flipCtrlCmd = mac && (qtwebkit || presto && (presto_version == null || presto_version < 12.11))
let captureRightClick = gecko || (ie && ie_version >= 9)


/***/ }),

/***/ "./node_modules/codemirror/src/util/dom.js":
/*!*************************************************!*\
  !*** ./node_modules/codemirror/src/util/dom.js ***!
  \*************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "classTest": () => (/* binding */ classTest),
/* harmony export */   "rmClass": () => (/* binding */ rmClass),
/* harmony export */   "removeChildren": () => (/* binding */ removeChildren),
/* harmony export */   "removeChildrenAndAdd": () => (/* binding */ removeChildrenAndAdd),
/* harmony export */   "elt": () => (/* binding */ elt),
/* harmony export */   "eltP": () => (/* binding */ eltP),
/* harmony export */   "range": () => (/* binding */ range),
/* harmony export */   "contains": () => (/* binding */ contains),
/* harmony export */   "activeElt": () => (/* binding */ activeElt),
/* harmony export */   "addClass": () => (/* binding */ addClass),
/* harmony export */   "joinClasses": () => (/* binding */ joinClasses),
/* harmony export */   "selectInput": () => (/* binding */ selectInput)
/* harmony export */ });
/* harmony import */ var _browser_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./browser.js */ "./node_modules/codemirror/src/util/browser.js");


function classTest(cls) { return new RegExp("(^|\\s)" + cls + "(?:$|\\s)\\s*") }

let rmClass = function(node, cls) {
  let current = node.className
  let match = classTest(cls).exec(current)
  if (match) {
    let after = current.slice(match.index + match[0].length)
    node.className = current.slice(0, match.index) + (after ? match[1] + after : "")
  }
}

function removeChildren(e) {
  for (let count = e.childNodes.length; count > 0; --count)
    e.removeChild(e.firstChild)
  return e
}

function removeChildrenAndAdd(parent, e) {
  return removeChildren(parent).appendChild(e)
}

function elt(tag, content, className, style) {
  let e = document.createElement(tag)
  if (className) e.className = className
  if (style) e.style.cssText = style
  if (typeof content == "string") e.appendChild(document.createTextNode(content))
  else if (content) for (let i = 0; i < content.length; ++i) e.appendChild(content[i])
  return e
}
// wrapper for elt, which removes the elt from the accessibility tree
function eltP(tag, content, className, style) {
  let e = elt(tag, content, className, style)
  e.setAttribute("role", "presentation")
  return e
}

let range
if (document.createRange) range = function(node, start, end, endNode) {
  let r = document.createRange()
  r.setEnd(endNode || node, end)
  r.setStart(node, start)
  return r
}
else range = function(node, start, end) {
  let r = document.body.createTextRange()
  try { r.moveToElementText(node.parentNode) }
  catch(e) { return r }
  r.collapse(true)
  r.moveEnd("character", end)
  r.moveStart("character", start)
  return r
}

function contains(parent, child) {
  if (child.nodeType == 3) // Android browser always returns false when child is a textnode
    child = child.parentNode
  if (parent.contains)
    return parent.contains(child)
  do {
    if (child.nodeType == 11) child = child.host
    if (child == parent) return true
  } while (child = child.parentNode)
}

function activeElt() {
  // IE and Edge may throw an "Unspecified Error" when accessing document.activeElement.
  // IE < 10 will throw when accessed while the page is loading or in an iframe.
  // IE > 9 and Edge will throw when accessed in an iframe if document.body is unavailable.
  let activeElement
  try {
    activeElement = document.activeElement
  } catch(e) {
    activeElement = document.body || null
  }
  while (activeElement && activeElement.shadowRoot && activeElement.shadowRoot.activeElement)
    activeElement = activeElement.shadowRoot.activeElement
  return activeElement
}

function addClass(node, cls) {
  let current = node.className
  if (!classTest(cls).test(current)) node.className += (current ? " " : "") + cls
}
function joinClasses(a, b) {
  let as = a.split(" ")
  for (let i = 0; i < as.length; i++)
    if (as[i] && !classTest(as[i]).test(b)) b += " " + as[i]
  return b
}

let selectInput = function(node) { node.select() }
if (_browser_js__WEBPACK_IMPORTED_MODULE_0__.ios) // Mobile Safari apparently has a bug where select() is broken.
  selectInput = function(node) { node.selectionStart = 0; node.selectionEnd = node.value.length }
else if (_browser_js__WEBPACK_IMPORTED_MODULE_0__.ie) // Suppress mysterious IE10 errors
  selectInput = function(node) { try { node.select() } catch(_e) {} }


/***/ }),

/***/ "./node_modules/codemirror/src/util/event.js":
/*!***************************************************!*\
  !*** ./node_modules/codemirror/src/util/event.js ***!
  \***************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "on": () => (/* binding */ on),
/* harmony export */   "getHandlers": () => (/* binding */ getHandlers),
/* harmony export */   "off": () => (/* binding */ off),
/* harmony export */   "signal": () => (/* binding */ signal),
/* harmony export */   "signalDOMEvent": () => (/* binding */ signalDOMEvent),
/* harmony export */   "signalCursorActivity": () => (/* binding */ signalCursorActivity),
/* harmony export */   "hasHandler": () => (/* binding */ hasHandler),
/* harmony export */   "eventMixin": () => (/* binding */ eventMixin),
/* harmony export */   "e_preventDefault": () => (/* binding */ e_preventDefault),
/* harmony export */   "e_stopPropagation": () => (/* binding */ e_stopPropagation),
/* harmony export */   "e_defaultPrevented": () => (/* binding */ e_defaultPrevented),
/* harmony export */   "e_stop": () => (/* binding */ e_stop),
/* harmony export */   "e_target": () => (/* binding */ e_target),
/* harmony export */   "e_button": () => (/* binding */ e_button)
/* harmony export */ });
/* harmony import */ var _browser_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./browser.js */ "./node_modules/codemirror/src/util/browser.js");
/* harmony import */ var _misc_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ./misc.js */ "./node_modules/codemirror/src/util/misc.js");



// EVENT HANDLING

// Lightweight event framework. on/off also work on DOM nodes,
// registering native DOM handlers.

const noHandlers = []

let on = function(emitter, type, f) {
  if (emitter.addEventListener) {
    emitter.addEventListener(type, f, false)
  } else if (emitter.attachEvent) {
    emitter.attachEvent("on" + type, f)
  } else {
    let map = emitter._handlers || (emitter._handlers = {})
    map[type] = (map[type] || noHandlers).concat(f)
  }
}

function getHandlers(emitter, type) {
  return emitter._handlers && emitter._handlers[type] || noHandlers
}

function off(emitter, type, f) {
  if (emitter.removeEventListener) {
    emitter.removeEventListener(type, f, false)
  } else if (emitter.detachEvent) {
    emitter.detachEvent("on" + type, f)
  } else {
    let map = emitter._handlers, arr = map && map[type]
    if (arr) {
      let index = (0,_misc_js__WEBPACK_IMPORTED_MODULE_1__.indexOf)(arr, f)
      if (index > -1)
        map[type] = arr.slice(0, index).concat(arr.slice(index + 1))
    }
  }
}

function signal(emitter, type /*, values...*/) {
  let handlers = getHandlers(emitter, type)
  if (!handlers.length) return
  let args = Array.prototype.slice.call(arguments, 2)
  for (let i = 0; i < handlers.length; ++i) handlers[i].apply(null, args)
}

// The DOM events that CodeMirror handles can be overridden by
// registering a (non-DOM) handler on the editor for the event name,
// and preventDefault-ing the event in that handler.
function signalDOMEvent(cm, e, override) {
  if (typeof e == "string")
    e = {type: e, preventDefault: function() { this.defaultPrevented = true }}
  signal(cm, override || e.type, cm, e)
  return e_defaultPrevented(e) || e.codemirrorIgnore
}

function signalCursorActivity(cm) {
  let arr = cm._handlers && cm._handlers.cursorActivity
  if (!arr) return
  let set = cm.curOp.cursorActivityHandlers || (cm.curOp.cursorActivityHandlers = [])
  for (let i = 0; i < arr.length; ++i) if ((0,_misc_js__WEBPACK_IMPORTED_MODULE_1__.indexOf)(set, arr[i]) == -1)
    set.push(arr[i])
}

function hasHandler(emitter, type) {
  return getHandlers(emitter, type).length > 0
}

// Add on and off methods to a constructor's prototype, to make
// registering events on such objects more convenient.
function eventMixin(ctor) {
  ctor.prototype.on = function(type, f) {on(this, type, f)}
  ctor.prototype.off = function(type, f) {off(this, type, f)}
}

// Due to the fact that we still support jurassic IE versions, some
// compatibility wrappers are needed.

function e_preventDefault(e) {
  if (e.preventDefault) e.preventDefault()
  else e.returnValue = false
}
function e_stopPropagation(e) {
  if (e.stopPropagation) e.stopPropagation()
  else e.cancelBubble = true
}
function e_defaultPrevented(e) {
  return e.defaultPrevented != null ? e.defaultPrevented : e.returnValue == false
}
function e_stop(e) {e_preventDefault(e); e_stopPropagation(e)}

function e_target(e) {return e.target || e.srcElement}
function e_button(e) {
  let b = e.which
  if (b == null) {
    if (e.button & 1) b = 1
    else if (e.button & 2) b = 3
    else if (e.button & 4) b = 2
  }
  if (_browser_js__WEBPACK_IMPORTED_MODULE_0__.mac && e.ctrlKey && b == 1) b = 3
  return b
}


/***/ }),

/***/ "./node_modules/codemirror/src/util/feature_detection.js":
/*!***************************************************************!*\
  !*** ./node_modules/codemirror/src/util/feature_detection.js ***!
  \***************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "dragAndDrop": () => (/* binding */ dragAndDrop),
/* harmony export */   "zeroWidthElement": () => (/* binding */ zeroWidthElement),
/* harmony export */   "hasBadBidiRects": () => (/* binding */ hasBadBidiRects),
/* harmony export */   "splitLinesAuto": () => (/* binding */ splitLinesAuto),
/* harmony export */   "hasSelection": () => (/* binding */ hasSelection),
/* harmony export */   "hasCopyEvent": () => (/* binding */ hasCopyEvent),
/* harmony export */   "hasBadZoomedRects": () => (/* binding */ hasBadZoomedRects)
/* harmony export */ });
/* harmony import */ var _dom_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./dom.js */ "./node_modules/codemirror/src/util/dom.js");
/* harmony import */ var _browser_js__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! ./browser.js */ "./node_modules/codemirror/src/util/browser.js");



// Detect drag-and-drop
let dragAndDrop = function() {
  // There is *some* kind of drag-and-drop support in IE6-8, but I
  // couldn't get it to work yet.
  if (_browser_js__WEBPACK_IMPORTED_MODULE_1__.ie && _browser_js__WEBPACK_IMPORTED_MODULE_1__.ie_version < 9) return false
  let div = (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)('div')
  return "draggable" in div || "dragDrop" in div
}()

let zwspSupported
function zeroWidthElement(measure) {
  if (zwspSupported == null) {
    let test = (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("span", "\u200b")
    ;(0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.removeChildrenAndAdd)(measure, (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("span", [test, document.createTextNode("x")]))
    if (measure.firstChild.offsetHeight != 0)
      zwspSupported = test.offsetWidth <= 1 && test.offsetHeight > 2 && !(_browser_js__WEBPACK_IMPORTED_MODULE_1__.ie && _browser_js__WEBPACK_IMPORTED_MODULE_1__.ie_version < 8)
  }
  let node = zwspSupported ? (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("span", "\u200b") :
    (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("span", "\u00a0", null, "display: inline-block; width: 1px; margin-right: -1px")
  node.setAttribute("cm-text", "")
  return node
}

// Feature-detect IE's crummy client rect reporting for bidi text
let badBidiRects
function hasBadBidiRects(measure) {
  if (badBidiRects != null) return badBidiRects
  let txt = (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.removeChildrenAndAdd)(measure, document.createTextNode("A\u062eA"))
  let r0 = (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.range)(txt, 0, 1).getBoundingClientRect()
  let r1 = (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.range)(txt, 1, 2).getBoundingClientRect()
  ;(0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.removeChildren)(measure)
  if (!r0 || r0.left == r0.right) return false // Safari returns null in some cases (#2780)
  return badBidiRects = (r1.right - r0.right < 3)
}

// See if "".split is the broken IE version, if so, provide an
// alternative way to split lines.
let splitLinesAuto = "\n\nb".split(/\n/).length != 3 ? string => {
  let pos = 0, result = [], l = string.length
  while (pos <= l) {
    let nl = string.indexOf("\n", pos)
    if (nl == -1) nl = string.length
    let line = string.slice(pos, string.charAt(nl - 1) == "\r" ? nl - 1 : nl)
    let rt = line.indexOf("\r")
    if (rt != -1) {
      result.push(line.slice(0, rt))
      pos += rt + 1
    } else {
      result.push(line)
      pos = nl + 1
    }
  }
  return result
} : string => string.split(/\r\n?|\n/)

let hasSelection = window.getSelection ? te => {
  try { return te.selectionStart != te.selectionEnd }
  catch(e) { return false }
} : te => {
  let range
  try {range = te.ownerDocument.selection.createRange()}
  catch(e) {}
  if (!range || range.parentElement() != te) return false
  return range.compareEndPoints("StartToEnd", range) != 0
}

let hasCopyEvent = (() => {
  let e = (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("div")
  if ("oncopy" in e) return true
  e.setAttribute("oncopy", "return;")
  return typeof e.oncopy == "function"
})()

let badZoomedRects = null
function hasBadZoomedRects(measure) {
  if (badZoomedRects != null) return badZoomedRects
  let node = (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.removeChildrenAndAdd)(measure, (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.elt)("span", "x"))
  let normal = node.getBoundingClientRect()
  let fromRange = (0,_dom_js__WEBPACK_IMPORTED_MODULE_0__.range)(node, 0, 1).getBoundingClientRect()
  return badZoomedRects = Math.abs(normal.left - fromRange.left) > 1
}


/***/ }),

/***/ "./node_modules/codemirror/src/util/misc.js":
/*!**************************************************!*\
  !*** ./node_modules/codemirror/src/util/misc.js ***!
  \**************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "bind": () => (/* binding */ bind),
/* harmony export */   "copyObj": () => (/* binding */ copyObj),
/* harmony export */   "countColumn": () => (/* binding */ countColumn),
/* harmony export */   "Delayed": () => (/* binding */ Delayed),
/* harmony export */   "indexOf": () => (/* binding */ indexOf),
/* harmony export */   "scrollerGap": () => (/* binding */ scrollerGap),
/* harmony export */   "Pass": () => (/* binding */ Pass),
/* harmony export */   "sel_dontScroll": () => (/* binding */ sel_dontScroll),
/* harmony export */   "sel_mouse": () => (/* binding */ sel_mouse),
/* harmony export */   "sel_move": () => (/* binding */ sel_move),
/* harmony export */   "findColumn": () => (/* binding */ findColumn),
/* harmony export */   "spaceStr": () => (/* binding */ spaceStr),
/* harmony export */   "lst": () => (/* binding */ lst),
/* harmony export */   "map": () => (/* binding */ map),
/* harmony export */   "insertSorted": () => (/* binding */ insertSorted),
/* harmony export */   "createObj": () => (/* binding */ createObj),
/* harmony export */   "isWordCharBasic": () => (/* binding */ isWordCharBasic),
/* harmony export */   "isWordChar": () => (/* binding */ isWordChar),
/* harmony export */   "isEmpty": () => (/* binding */ isEmpty),
/* harmony export */   "isExtendingChar": () => (/* binding */ isExtendingChar),
/* harmony export */   "skipExtendingChars": () => (/* binding */ skipExtendingChars),
/* harmony export */   "findFirst": () => (/* binding */ findFirst)
/* harmony export */ });
function bind(f) {
  let args = Array.prototype.slice.call(arguments, 1)
  return function(){return f.apply(null, args)}
}

function copyObj(obj, target, overwrite) {
  if (!target) target = {}
  for (let prop in obj)
    if (obj.hasOwnProperty(prop) && (overwrite !== false || !target.hasOwnProperty(prop)))
      target[prop] = obj[prop]
  return target
}

// Counts the column offset in a string, taking tabs into account.
// Used mostly to find indentation.
function countColumn(string, end, tabSize, startIndex, startValue) {
  if (end == null) {
    end = string.search(/[^\s\u00a0]/)
    if (end == -1) end = string.length
  }
  for (let i = startIndex || 0, n = startValue || 0;;) {
    let nextTab = string.indexOf("\t", i)
    if (nextTab < 0 || nextTab >= end)
      return n + (end - i)
    n += nextTab - i
    n += tabSize - (n % tabSize)
    i = nextTab + 1
  }
}

class Delayed {
  constructor() {
    this.id = null
    this.f = null
    this.time = 0
    this.handler = bind(this.onTimeout, this)
  }
  onTimeout(self) {
    self.id = 0
    if (self.time <= +new Date) {
      self.f()
    } else {
      setTimeout(self.handler, self.time - +new Date)
    }
  }
  set(ms, f) {
    this.f = f
    const time = +new Date + ms
    if (!this.id || time < this.time) {
      clearTimeout(this.id)
      this.id = setTimeout(this.handler, ms)
      this.time = time
    }
  }
}

function indexOf(array, elt) {
  for (let i = 0; i < array.length; ++i)
    if (array[i] == elt) return i
  return -1
}

// Number of pixels added to scroller and sizer to hide scrollbar
let scrollerGap = 50

// Returned or thrown by various protocols to signal 'I'm not
// handling this'.
let Pass = {toString: function(){return "CodeMirror.Pass"}}

// Reused option objects for setSelection & friends
let sel_dontScroll = {scroll: false}, sel_mouse = {origin: "*mouse"}, sel_move = {origin: "+move"}

// The inverse of countColumn -- find the offset that corresponds to
// a particular column.
function findColumn(string, goal, tabSize) {
  for (let pos = 0, col = 0;;) {
    let nextTab = string.indexOf("\t", pos)
    if (nextTab == -1) nextTab = string.length
    let skipped = nextTab - pos
    if (nextTab == string.length || col + skipped >= goal)
      return pos + Math.min(skipped, goal - col)
    col += nextTab - pos
    col += tabSize - (col % tabSize)
    pos = nextTab + 1
    if (col >= goal) return pos
  }
}

let spaceStrs = [""]
function spaceStr(n) {
  while (spaceStrs.length <= n)
    spaceStrs.push(lst(spaceStrs) + " ")
  return spaceStrs[n]
}

function lst(arr) { return arr[arr.length-1] }

function map(array, f) {
  let out = []
  for (let i = 0; i < array.length; i++) out[i] = f(array[i], i)
  return out
}

function insertSorted(array, value, score) {
  let pos = 0, priority = score(value)
  while (pos < array.length && score(array[pos]) <= priority) pos++
  array.splice(pos, 0, value)
}

function nothing() {}

function createObj(base, props) {
  let inst
  if (Object.create) {
    inst = Object.create(base)
  } else {
    nothing.prototype = base
    inst = new nothing()
  }
  if (props) copyObj(props, inst)
  return inst
}

let nonASCIISingleCaseWordChar = /[\u00df\u0587\u0590-\u05f4\u0600-\u06ff\u3040-\u309f\u30a0-\u30ff\u3400-\u4db5\u4e00-\u9fcc\uac00-\ud7af]/
function isWordCharBasic(ch) {
  return /\w/.test(ch) || ch > "\x80" &&
    (ch.toUpperCase() != ch.toLowerCase() || nonASCIISingleCaseWordChar.test(ch))
}
function isWordChar(ch, helper) {
  if (!helper) return isWordCharBasic(ch)
  if (helper.source.indexOf("\\w") > -1 && isWordCharBasic(ch)) return true
  return helper.test(ch)
}

function isEmpty(obj) {
  for (let n in obj) if (obj.hasOwnProperty(n) && obj[n]) return false
  return true
}

// Extending unicode characters. A series of a non-extending char +
// any number of extending chars is treated as a single unit as far
// as editing and measuring is concerned. This is not fully correct,
// since some scripts/fonts/browsers also treat other configurations
// of code points as a group.
let extendingChars = /[\u0300-\u036f\u0483-\u0489\u0591-\u05bd\u05bf\u05c1\u05c2\u05c4\u05c5\u05c7\u0610-\u061a\u064b-\u065e\u0670\u06d6-\u06dc\u06de-\u06e4\u06e7\u06e8\u06ea-\u06ed\u0711\u0730-\u074a\u07a6-\u07b0\u07eb-\u07f3\u0816-\u0819\u081b-\u0823\u0825-\u0827\u0829-\u082d\u0900-\u0902\u093c\u0941-\u0948\u094d\u0951-\u0955\u0962\u0963\u0981\u09bc\u09be\u09c1-\u09c4\u09cd\u09d7\u09e2\u09e3\u0a01\u0a02\u0a3c\u0a41\u0a42\u0a47\u0a48\u0a4b-\u0a4d\u0a51\u0a70\u0a71\u0a75\u0a81\u0a82\u0abc\u0ac1-\u0ac5\u0ac7\u0ac8\u0acd\u0ae2\u0ae3\u0b01\u0b3c\u0b3e\u0b3f\u0b41-\u0b44\u0b4d\u0b56\u0b57\u0b62\u0b63\u0b82\u0bbe\u0bc0\u0bcd\u0bd7\u0c3e-\u0c40\u0c46-\u0c48\u0c4a-\u0c4d\u0c55\u0c56\u0c62\u0c63\u0cbc\u0cbf\u0cc2\u0cc6\u0ccc\u0ccd\u0cd5\u0cd6\u0ce2\u0ce3\u0d3e\u0d41-\u0d44\u0d4d\u0d57\u0d62\u0d63\u0dca\u0dcf\u0dd2-\u0dd4\u0dd6\u0ddf\u0e31\u0e34-\u0e3a\u0e47-\u0e4e\u0eb1\u0eb4-\u0eb9\u0ebb\u0ebc\u0ec8-\u0ecd\u0f18\u0f19\u0f35\u0f37\u0f39\u0f71-\u0f7e\u0f80-\u0f84\u0f86\u0f87\u0f90-\u0f97\u0f99-\u0fbc\u0fc6\u102d-\u1030\u1032-\u1037\u1039\u103a\u103d\u103e\u1058\u1059\u105e-\u1060\u1071-\u1074\u1082\u1085\u1086\u108d\u109d\u135f\u1712-\u1714\u1732-\u1734\u1752\u1753\u1772\u1773\u17b7-\u17bd\u17c6\u17c9-\u17d3\u17dd\u180b-\u180d\u18a9\u1920-\u1922\u1927\u1928\u1932\u1939-\u193b\u1a17\u1a18\u1a56\u1a58-\u1a5e\u1a60\u1a62\u1a65-\u1a6c\u1a73-\u1a7c\u1a7f\u1b00-\u1b03\u1b34\u1b36-\u1b3a\u1b3c\u1b42\u1b6b-\u1b73\u1b80\u1b81\u1ba2-\u1ba5\u1ba8\u1ba9\u1c2c-\u1c33\u1c36\u1c37\u1cd0-\u1cd2\u1cd4-\u1ce0\u1ce2-\u1ce8\u1ced\u1dc0-\u1de6\u1dfd-\u1dff\u200c\u200d\u20d0-\u20f0\u2cef-\u2cf1\u2de0-\u2dff\u302a-\u302f\u3099\u309a\ua66f-\ua672\ua67c\ua67d\ua6f0\ua6f1\ua802\ua806\ua80b\ua825\ua826\ua8c4\ua8e0-\ua8f1\ua926-\ua92d\ua947-\ua951\ua980-\ua982\ua9b3\ua9b6-\ua9b9\ua9bc\uaa29-\uaa2e\uaa31\uaa32\uaa35\uaa36\uaa43\uaa4c\uaab0\uaab2-\uaab4\uaab7\uaab8\uaabe\uaabf\uaac1\uabe5\uabe8\uabed\udc00-\udfff\ufb1e\ufe00-\ufe0f\ufe20-\ufe26\uff9e\uff9f]/
function isExtendingChar(ch) { return ch.charCodeAt(0) >= 768 && extendingChars.test(ch) }

// Returns a number from the range [`0`; `str.length`] unless `pos` is outside that range.
function skipExtendingChars(str, pos, dir) {
  while ((dir < 0 ? pos > 0 : pos < str.length) && isExtendingChar(str.charAt(pos))) pos += dir
  return pos
}

// Returns the value from the range [`from`; `to`] that satisfies
// `pred` and is closest to `from`. Assumes that at least `to`
// satisfies `pred`. Supports `from` being greater than `to`.
function findFirst(pred, from, to) {
  // At any point we are certain `to` satisfies `pred`, don't know
  // whether `from` does.
  let dir = from > to ? -1 : 1
  for (;;) {
    if (from == to) return from
    let midF = (from + to) / 2, mid = dir < 0 ? Math.ceil(midF) : Math.floor(midF)
    if (mid == from) return pred(mid) ? from : to
    if (pred(mid)) to = mid
    else from = mid + dir
  }
}


/***/ }),

/***/ "./node_modules/codemirror/src/util/operation_group.js":
/*!*************************************************************!*\
  !*** ./node_modules/codemirror/src/util/operation_group.js ***!
  \*************************************************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "pushOperation": () => (/* binding */ pushOperation),
/* harmony export */   "finishOperation": () => (/* binding */ finishOperation),
/* harmony export */   "signalLater": () => (/* binding */ signalLater)
/* harmony export */ });
/* harmony import */ var _event_js__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./event.js */ "./node_modules/codemirror/src/util/event.js");


let operationGroup = null

function pushOperation(op) {
  if (operationGroup) {
    operationGroup.ops.push(op)
  } else {
    op.ownsGroup = operationGroup = {
      ops: [op],
      delayedCallbacks: []
    }
  }
}

function fireCallbacksForOps(group) {
  // Calls delayed callbacks and cursorActivity handlers until no
  // new ones appear
  let callbacks = group.delayedCallbacks, i = 0
  do {
    for (; i < callbacks.length; i++)
      callbacks[i].call(null)
    for (let j = 0; j < group.ops.length; j++) {
      let op = group.ops[j]
      if (op.cursorActivityHandlers)
        while (op.cursorActivityCalled < op.cursorActivityHandlers.length)
          op.cursorActivityHandlers[op.cursorActivityCalled++].call(null, op.cm)
    }
  } while (i < callbacks.length)
}

function finishOperation(op, endCb) {
  let group = op.ownsGroup
  if (!group) return

  try { fireCallbacksForOps(group) }
  finally {
    operationGroup = null
    endCb(group)
  }
}

let orphanDelayedCallbacks = null

// Often, we want to signal events at a point where we are in the
// middle of some work, but don't want the handler to start calling
// other methods on the editor, which might be in an inconsistent
// state or simply not expect any other events to happen.
// signalLater looks whether there are any handlers, and schedules
// them to be executed when the last operation ends, or, if no
// operation is active, when a timeout fires.
function signalLater(emitter, type /*, values...*/) {
  let arr = (0,_event_js__WEBPACK_IMPORTED_MODULE_0__.getHandlers)(emitter, type)
  if (!arr.length) return
  let args = Array.prototype.slice.call(arguments, 2), list
  if (operationGroup) {
    list = operationGroup.delayedCallbacks
  } else if (orphanDelayedCallbacks) {
    list = orphanDelayedCallbacks
  } else {
    list = orphanDelayedCallbacks = []
    setTimeout(fireOrphanDelayed, 0)
  }
  for (let i = 0; i < arr.length; ++i)
    list.push(() => arr[i].apply(null, args))
}

function fireOrphanDelayed() {
  let delayed = orphanDelayedCallbacks
  orphanDelayedCallbacks = null
  for (let i = 0; i < delayed.length; ++i) delayed[i]()
}


/***/ }),

/***/ "./node_modules/copy-to-clipboard/index.js":
/*!*************************************************!*\
  !*** ./node_modules/copy-to-clipboard/index.js ***!
  \*************************************************/
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

"use strict";


var deselectCurrent = __webpack_require__(/*! toggle-selection */ "./node_modules/toggle-selection/index.js");

var clipboardToIE11Formatting = {
  "text/plain": "Text",
  "text/html": "Url",
  "default": "Text"
}

var defaultMessage = "Copy to clipboard: #{key}, Enter";

function format(message) {
  var copyKey = (/mac os x/i.test(navigator.userAgent) ? "⌘" : "Ctrl") + "+C";
  return message.replace(/#{\s*key\s*}/g, copyKey);
}

function copy(text, options) {
  var debug,
    message,
    reselectPrevious,
    range,
    selection,
    mark,
    success = false;
  if (!options) {
    options = {};
  }
  debug = options.debug || false;
  try {
    reselectPrevious = deselectCurrent();

    range = document.createRange();
    selection = document.getSelection();

    mark = document.createElement("span");
    mark.textContent = text;
    // reset user styles for span element
    mark.style.all = "unset";
    // prevents scrolling to the end of the page
    mark.style.position = "fixed";
    mark.style.top = 0;
    mark.style.clip = "rect(0, 0, 0, 0)";
    // used to preserve spaces and line breaks
    mark.style.whiteSpace = "pre";
    // do not inherit user-select (it may be `none`)
    mark.style.webkitUserSelect = "text";
    mark.style.MozUserSelect = "text";
    mark.style.msUserSelect = "text";
    mark.style.userSelect = "text";
    mark.addEventListener("copy", function(e) {
      e.stopPropagation();
      if (options.format) {
        e.preventDefault();
        if (typeof e.clipboardData === "undefined") { // IE 11
          debug && console.warn("unable to use e.clipboardData");
          debug && console.warn("trying IE specific stuff");
          window.clipboardData.clearData();
          var format = clipboardToIE11Formatting[options.format] || clipboardToIE11Formatting["default"]
          window.clipboardData.setData(format, text);
        } else { // all other browsers
          e.clipboardData.clearData();
          e.clipboardData.setData(options.format, text);
        }
      }
      if (options.onCopy) {
        e.preventDefault();
        options.onCopy(e.clipboardData);
      }
    });

    document.body.appendChild(mark);

    range.selectNodeContents(mark);
    selection.addRange(range);

    var successful = document.execCommand("copy");
    if (!successful) {
      throw new Error("copy command was unsuccessful");
    }
    success = true;
  } catch (err) {
    debug && console.error("unable to copy using execCommand: ", err);
    debug && console.warn("trying IE specific stuff");
    try {
      window.clipboardData.setData(options.format || "text", text);
      options.onCopy && options.onCopy(window.clipboardData);
      success = true;
    } catch (err) {
      debug && console.error("unable to copy using clipboardData: ", err);
      debug && console.error("falling back to prompt");
      message = format("message" in options ? options.message : defaultMessage);
      window.prompt(message, text);
    }
  } finally {
    if (selection) {
      if (typeof selection.removeRange == "function") {
        selection.removeRange(range);
      } else {
        selection.removeAllRanges();
      }
    }

    if (mark) {
      document.body.removeChild(mark);
    }
    reselectPrevious();
  }

  return success;
}

module.exports = copy;


/***/ }),

/***/ "./node_modules/izitoast/dist/js/iziToast.js":
/*!***************************************************!*\
  !*** ./node_modules/izitoast/dist/js/iziToast.js ***!
  \***************************************************/
/***/ (function(module, exports, __webpack_require__) {

var __WEBPACK_AMD_DEFINE_FACTORY__, __WEBPACK_AMD_DEFINE_ARRAY__, __WEBPACK_AMD_DEFINE_RESULT__;/*
* iziToast | v1.4.0
* http://izitoast.marcelodolce.com
* by Marcelo Dolce.
*/
(function (root, factory) {
	if(true) {
		!(__WEBPACK_AMD_DEFINE_ARRAY__ = [], __WEBPACK_AMD_DEFINE_FACTORY__ = (factory(root)),
		__WEBPACK_AMD_DEFINE_RESULT__ = (typeof __WEBPACK_AMD_DEFINE_FACTORY__ === 'function' ?
		(__WEBPACK_AMD_DEFINE_FACTORY__.apply(exports, __WEBPACK_AMD_DEFINE_ARRAY__)) : __WEBPACK_AMD_DEFINE_FACTORY__),
		__WEBPACK_AMD_DEFINE_RESULT__ !== undefined && (module.exports = __WEBPACK_AMD_DEFINE_RESULT__));
	} else {}
})(typeof __webpack_require__.g !== 'undefined' ? __webpack_require__.g : window || this.window || this.global, function (root) {

	'use strict';

	//
	// Variables
	//
	var $iziToast = {},
		PLUGIN_NAME = 'iziToast',
		BODY = document.querySelector('body'),
		ISMOBILE = (/Mobi/.test(navigator.userAgent)) ? true : false,
		ISCHROME = /Chrome/.test(navigator.userAgent) && /Google Inc/.test(navigator.vendor),
		ISFIREFOX = typeof InstallTrigger !== 'undefined',
		ACCEPTSTOUCH = 'ontouchstart' in document.documentElement,
		POSITIONS = ['bottomRight','bottomLeft','bottomCenter','topRight','topLeft','topCenter','center'],
		THEMES = {
			info: {
				color: 'blue',
				icon: 'ico-info'
			},
			success: {
				color: 'green',
				icon: 'ico-success'
			},
			warning: {
				color: 'orange',
				icon: 'ico-warning'
			},
			error: {
				color: 'red',
				icon: 'ico-error'
			},
			question: {
				color: 'yellow',
				icon: 'ico-question'
			}
		},
		MOBILEWIDTH = 568,
		CONFIG = {};

	$iziToast.children = {};

	// Default settings
	var defaults = {
		id: null, 
		class: '',
		title: '',
		titleColor: '',
		titleSize: '',
		titleLineHeight: '',
		message: '',
		messageColor: '',
		messageSize: '',
		messageLineHeight: '',
		backgroundColor: '',
		theme: 'light', // dark
		color: '', // blue, red, green, yellow
		icon: '',
		iconText: '',
		iconColor: '',
		iconUrl: null,
		image: '',
		imageWidth: 50,
		maxWidth: null,
		zindex: null,
		layout: 1,
		balloon: false,
		close: true,
		closeOnEscape: false,
		closeOnClick: false,
		displayMode: 0,
		position: 'bottomRight', // bottomRight, bottomLeft, topRight, topLeft, topCenter, bottomCenter, center
		target: '',
		targetFirst: true,
		timeout: 5000,
		rtl: false,
		animateInside: true,
		drag: true,
		pauseOnHover: true,
		resetOnHover: false,
		progressBar: true,
		progressBarColor: '',
		progressBarEasing: 'linear',
		overlay: false,
		overlayClose: false,
		overlayColor: 'rgba(0, 0, 0, 0.6)',
		transitionIn: 'fadeInUp', // bounceInLeft, bounceInRight, bounceInUp, bounceInDown, fadeIn, fadeInDown, fadeInUp, fadeInLeft, fadeInRight, flipInX
		transitionOut: 'fadeOut', // fadeOut, fadeOutUp, fadeOutDown, fadeOutLeft, fadeOutRight, flipOutX
		transitionInMobile: 'fadeInUp',
		transitionOutMobile: 'fadeOutDown',
		buttons: {},
		inputs: {},
		onOpening: function () {},
		onOpened: function () {},
		onClosing: function () {},
		onClosed: function () {}
	};

	//
	// Methods
	//


	/**
	 * Polyfill for remove() method
	 */
	if(!('remove' in Element.prototype)) {
	    Element.prototype.remove = function() {
	        if(this.parentNode) {
	            this.parentNode.removeChild(this);
	        }
	    };
	}

	/*
     * Polyfill for CustomEvent for IE >= 9
     * https://developer.mozilla.org/en-US/docs/Web/API/CustomEvent/CustomEvent#Polyfill
     */
    if(typeof window.CustomEvent !== 'function') {
        var CustomEventPolyfill = function (event, params) {
            params = params || { bubbles: false, cancelable: false, detail: undefined };
            var evt = document.createEvent('CustomEvent');
            evt.initCustomEvent(event, params.bubbles, params.cancelable, params.detail);
            return evt;
        };

        CustomEventPolyfill.prototype = window.Event.prototype;

        window.CustomEvent = CustomEventPolyfill;
    }

	/**
	 * A simple forEach() implementation for Arrays, Objects and NodeLists
	 * @private
	 * @param {Array|Object|NodeList} collection Collection of items to iterate
	 * @param {Function} callback Callback function for each iteration
	 * @param {Array|Object|NodeList} scope Object/NodeList/Array that forEach is iterating over (aka `this`)
	 */
	var forEach = function (collection, callback, scope) {
		if(Object.prototype.toString.call(collection) === '[object Object]') {
			for (var prop in collection) {
				if(Object.prototype.hasOwnProperty.call(collection, prop)) {
					callback.call(scope, collection[prop], prop, collection);
				}
			}
		} else {
			if(collection){
				for (var i = 0, len = collection.length; i < len; i++) {
					callback.call(scope, collection[i], i, collection);
				}
			}
		}
	};

	/**
	 * Merge defaults with user options
	 * @private
	 * @param {Object} defaults Default settings
	 * @param {Object} options User options
	 * @returns {Object} Merged values of defaults and options
	 */
	var extend = function (defaults, options) {
		var extended = {};
		forEach(defaults, function (value, prop) {
			extended[prop] = defaults[prop];
		});
		forEach(options, function (value, prop) {
			extended[prop] = options[prop];
		});
		return extended;
	};


	/**
	 * Create a fragment DOM elements
	 * @private
	 */
	var createFragElem = function(htmlStr) {
		var frag = document.createDocumentFragment(),
			temp = document.createElement('div');
		temp.innerHTML = htmlStr;
		while (temp.firstChild) {
			frag.appendChild(temp.firstChild);
		}
		return frag;
	};


	/**
	 * Generate new ID
	 * @private
	 */
	var generateId = function(params) {
		var newId = btoa(encodeURIComponent(params));
		return newId.replace(/=/g, "");
	};


	/**
	 * Check if is a color
	 * @private
	 */
	var isColor = function(color){
		if( color.substring(0,1) == '#' || color.substring(0,3) == 'rgb' || color.substring(0,3) == 'hsl' ){
			return true;
		} else {
			return false;
		}
	};


	/**
	 * Check if is a Base64 string
	 * @private
	 */
	var isBase64 = function(str) {
	    try {
	        return btoa(atob(str)) == str;
	    } catch (err) {
	        return false;
	    }
	};


	/**
	 * Drag method of toasts
	 * @private
	 */
	var drag = function() {
	    
	    return {
	        move: function(toast, instance, settings, xpos) {

	        	var opacity,
	        		opacityRange = 0.3,
	        		distance = 180;
	            
	            if(xpos !== 0){
	            	
	            	toast.classList.add(PLUGIN_NAME+'-dragged');

	            	toast.style.transform = 'translateX('+xpos + 'px)';

		            if(xpos > 0){
		            	opacity = (distance-xpos) / distance;
		            	if(opacity < opacityRange){
							instance.hide(extend(settings, { transitionOut: 'fadeOutRight', transitionOutMobile: 'fadeOutRight' }), toast, 'drag');
						}
		            } else {
		            	opacity = (distance+xpos) / distance;
		            	if(opacity < opacityRange){
							instance.hide(extend(settings, { transitionOut: 'fadeOutLeft', transitionOutMobile: 'fadeOutLeft' }), toast, 'drag');
						}
		            }
					toast.style.opacity = opacity;
			
					if(opacity < opacityRange){

						if(ISCHROME || ISFIREFOX)
							toast.style.left = xpos+'px';

						toast.parentNode.style.opacity = opacityRange;

		                this.stopMoving(toast, null);
					}
	            }

				
	        },
	        startMoving: function(toast, instance, settings, e) {

	            e = e || window.event;
	            var posX = ((ACCEPTSTOUCH) ? e.touches[0].clientX : e.clientX),
	                toastLeft = toast.style.transform.replace('px)', '');
	                toastLeft = toastLeft.replace('translateX(', '');
	            var offsetX = posX - toastLeft;

				if(settings.transitionIn){
					toast.classList.remove(settings.transitionIn);
				}
				if(settings.transitionInMobile){
					toast.classList.remove(settings.transitionInMobile);
				}
				toast.style.transition = '';

	            if(ACCEPTSTOUCH) {
	                document.ontouchmove = function(e) {
	                    e.preventDefault();
	                    e = e || window.event;
	                    var posX = e.touches[0].clientX,
	                        finalX = posX - offsetX;
                        drag.move(toast, instance, settings, finalX);
	                };
	            } else {
	                document.onmousemove = function(e) {
	                    e.preventDefault();
	                    e = e || window.event;
	                    var posX = e.clientX,
	                        finalX = posX - offsetX;
                        drag.move(toast, instance, settings, finalX);
	                };
	            }

	        },
	        stopMoving: function(toast, e) {

	            if(ACCEPTSTOUCH) {
	                document.ontouchmove = function() {};
	            } else {
	            	document.onmousemove = function() {};
	            }

				toast.style.opacity = '';
				toast.style.transform = '';

	            if(toast.classList.contains(PLUGIN_NAME+'-dragged')){
	            	
	            	toast.classList.remove(PLUGIN_NAME+'-dragged');

					toast.style.transition = 'transform 0.4s ease, opacity 0.4s ease';
					setTimeout(function() {
						toast.style.transition = '';
					}, 400);
	            }

	        }
	    };

	}();





	$iziToast.setSetting = function (ref, option, value) {

		$iziToast.children[ref][option] = value;

	};


	$iziToast.getSetting = function (ref, option) {

		return $iziToast.children[ref][option];

	};


	/**
	 * Destroy the current initialization.
	 * @public
	 */
	$iziToast.destroy = function () {

		forEach(document.querySelectorAll('.'+PLUGIN_NAME+'-overlay'), function(element, index) {
			element.remove();
		});

		forEach(document.querySelectorAll('.'+PLUGIN_NAME+'-wrapper'), function(element, index) {
			element.remove();
		});

		forEach(document.querySelectorAll('.'+PLUGIN_NAME), function(element, index) {
			element.remove();
		});

		this.children = {};

		// Remove event listeners
		document.removeEventListener(PLUGIN_NAME+'-opened', {}, false);
		document.removeEventListener(PLUGIN_NAME+'-opening', {}, false);
		document.removeEventListener(PLUGIN_NAME+'-closing', {}, false);
		document.removeEventListener(PLUGIN_NAME+'-closed', {}, false);
		document.removeEventListener('keyup', {}, false);

		// Reset variables
		CONFIG = {};
	};

	/**
	 * Initialize Plugin
	 * @public
	 * @param {Object} options User settings
	 */
	$iziToast.settings = function (options) {

		// Destroy any existing initializations
		$iziToast.destroy();

		CONFIG = options;
		defaults = extend(defaults, options || {});
	};


	/**
	 * Building themes functions.
	 * @public
	 * @param {Object} options User settings
	 */
	forEach(THEMES, function (theme, name) {

		$iziToast[name] = function (options) {

			var settings = extend(CONFIG, options || {});
			settings = extend(theme, settings || {});

			this.show(settings);
		};

	});


	/**
	 * Do the calculation to move the progress bar
	 * @private
	 */
	$iziToast.progress = function (options, $toast, callback) {


		var that = this,
			ref = $toast.getAttribute('data-iziToast-ref'),
			settings = extend(this.children[ref], options || {}),
			$elem = $toast.querySelector('.'+PLUGIN_NAME+'-progressbar div');

	    return {
	        start: function() {

	        	if(typeof settings.time.REMAINING == 'undefined'){

	        		$toast.classList.remove(PLUGIN_NAME+'-reseted');

		        	if($elem !== null){
						$elem.style.transition = 'width '+ settings.timeout +'ms '+settings.progressBarEasing;
						$elem.style.width = '0%';
					}

		        	settings.time.START = new Date().getTime();
		        	settings.time.END = settings.time.START + settings.timeout;
					settings.time.TIMER = setTimeout(function() {

						clearTimeout(settings.time.TIMER);

						if(!$toast.classList.contains(PLUGIN_NAME+'-closing')){

							that.hide(settings, $toast, 'timeout');

							if(typeof callback === 'function'){
								callback.apply(that);
							}
						}

					}, settings.timeout);			
		        	that.setSetting(ref, 'time', settings.time);
	        	}
	        },
	        pause: function() {

	        	if(typeof settings.time.START !== 'undefined' && !$toast.classList.contains(PLUGIN_NAME+'-paused') && !$toast.classList.contains(PLUGIN_NAME+'-reseted')){

        			$toast.classList.add(PLUGIN_NAME+'-paused');

					settings.time.REMAINING = settings.time.END - new Date().getTime();

					clearTimeout(settings.time.TIMER);

					that.setSetting(ref, 'time', settings.time);

					if($elem !== null){
						var computedStyle = window.getComputedStyle($elem),
							propertyWidth = computedStyle.getPropertyValue('width');

						$elem.style.transition = 'none';
						$elem.style.width = propertyWidth;					
					}

					if(typeof callback === 'function'){
						setTimeout(function() {
							callback.apply(that);						
						}, 10);
					}
        		}
	        },
	        resume: function() {

				if(typeof settings.time.REMAINING !== 'undefined'){

					$toast.classList.remove(PLUGIN_NAME+'-paused');

		        	if($elem !== null){
						$elem.style.transition = 'width '+ settings.time.REMAINING +'ms '+settings.progressBarEasing;
						$elem.style.width = '0%';
					}

		        	settings.time.END = new Date().getTime() + settings.time.REMAINING;
					settings.time.TIMER = setTimeout(function() {

						clearTimeout(settings.time.TIMER);

						if(!$toast.classList.contains(PLUGIN_NAME+'-closing')){

							that.hide(settings, $toast, 'timeout');

							if(typeof callback === 'function'){
								callback.apply(that);
							}
						}


					}, settings.time.REMAINING);

					that.setSetting(ref, 'time', settings.time);
				} else {
					this.start();
				}
	        },
	        reset: function(){

				clearTimeout(settings.time.TIMER);

				delete settings.time.REMAINING;

				that.setSetting(ref, 'time', settings.time);

				$toast.classList.add(PLUGIN_NAME+'-reseted');

				$toast.classList.remove(PLUGIN_NAME+'-paused');

				if($elem !== null){
					$elem.style.transition = 'none';
					$elem.style.width = '100%';
				}

				if(typeof callback === 'function'){
					setTimeout(function() {
						callback.apply(that);						
					}, 10);
				}
	        }
	    };

	};


	/**
	 * Close the specific Toast
	 * @public
	 * @param {Object} options User settings
	 */
	$iziToast.hide = function (options, $toast, closedBy) {

		if(typeof $toast != 'object'){
			$toast = document.querySelector($toast);
		}		

		var that = this,
			settings = extend(this.children[$toast.getAttribute('data-iziToast-ref')], options || {});
			settings.closedBy = closedBy || null;

		delete settings.time.REMAINING;

		$toast.classList.add(PLUGIN_NAME+'-closing');

		// Overlay
		(function(){

			var $overlay = document.querySelector('.'+PLUGIN_NAME+'-overlay');
			if($overlay !== null){
				var refs = $overlay.getAttribute('data-iziToast-ref');		
					refs = refs.split(',');
				var index = refs.indexOf(String(settings.ref));

				if(index !== -1){
					refs.splice(index, 1);			
				}
				$overlay.setAttribute('data-iziToast-ref', refs.join());

				if(refs.length === 0){
					$overlay.classList.remove('fadeIn');
					$overlay.classList.add('fadeOut');
					setTimeout(function() {
						$overlay.remove();
					}, 700);
				}
			}

		})();

		if(settings.transitionIn){
			$toast.classList.remove(settings.transitionIn);
		} 

		if(settings.transitionInMobile){
			$toast.classList.remove(settings.transitionInMobile);
		}

		if(ISMOBILE || window.innerWidth <= MOBILEWIDTH){
			if(settings.transitionOutMobile)
				$toast.classList.add(settings.transitionOutMobile);
		} else {
			if(settings.transitionOut)
				$toast.classList.add(settings.transitionOut);
		}
		var H = $toast.parentNode.offsetHeight;
				$toast.parentNode.style.height = H+'px';
				$toast.style.pointerEvents = 'none';
		
		if(!ISMOBILE || window.innerWidth > MOBILEWIDTH){
			$toast.parentNode.style.transitionDelay = '0.2s';
		}

		try {
			var event = new CustomEvent(PLUGIN_NAME+'-closing', {detail: settings, bubbles: true, cancelable: true});
			document.dispatchEvent(event);
		} catch(ex){
			console.warn(ex);
		}

		setTimeout(function() {
			
			$toast.parentNode.style.height = '0px';
			$toast.parentNode.style.overflow = '';

			setTimeout(function(){
				
				delete that.children[settings.ref];

				$toast.parentNode.remove();

				try {
					var event = new CustomEvent(PLUGIN_NAME+'-closed', {detail: settings, bubbles: true, cancelable: true});
					document.dispatchEvent(event);
				} catch(ex){
					console.warn(ex);
				}

				if(typeof settings.onClosed !== 'undefined'){
					settings.onClosed.apply(null, [settings, $toast, closedBy]);
				}

			}, 1000);
		}, 200);


		if(typeof settings.onClosing !== 'undefined'){
			settings.onClosing.apply(null, [settings, $toast, closedBy]);
		}
	};

	/**
	 * Create and show the Toast
	 * @public
	 * @param {Object} options User settings
	 */
	$iziToast.show = function (options) {

		var that = this;

		// Merge user options with defaults
		var settings = extend(CONFIG, options || {});
			settings = extend(defaults, settings);
			settings.time = {};

		if(settings.id === null){
			settings.id = generateId(settings.title+settings.message+settings.color);
		}

		if(settings.displayMode === 1 || settings.displayMode == 'once'){
			try {
				if(document.querySelectorAll('.'+PLUGIN_NAME+'#'+settings.id).length > 0){
					return false;
				}
			} catch (exc) {
				console.warn('['+PLUGIN_NAME+'] Could not find an element with this selector: '+'#'+settings.id+'. Try to set an valid id.');
			}
		}

		if(settings.displayMode === 2 || settings.displayMode == 'replace'){
			try {
				forEach(document.querySelectorAll('.'+PLUGIN_NAME+'#'+settings.id), function(element, index) {
					that.hide(settings, element, 'replaced');
				});
			} catch (exc) {
				console.warn('['+PLUGIN_NAME+'] Could not find an element with this selector: '+'#'+settings.id+'. Try to set an valid id.');
			}
		}

		settings.ref = new Date().getTime() + Math.floor((Math.random() * 10000000) + 1);

		$iziToast.children[settings.ref] = settings;

		var $DOM = {
			body: document.querySelector('body'),
			overlay: document.createElement('div'),
			toast: document.createElement('div'),
			toastBody: document.createElement('div'),
			toastTexts: document.createElement('div'),
			toastCapsule: document.createElement('div'),
			cover: document.createElement('div'),
			buttons: document.createElement('div'),
			inputs: document.createElement('div'),
			icon: !settings.iconUrl ? document.createElement('i') : document.createElement('img'),
			wrapper: null
		};

		$DOM.toast.setAttribute('data-iziToast-ref', settings.ref);
		$DOM.toast.appendChild($DOM.toastBody);
		$DOM.toastCapsule.appendChild($DOM.toast);

		// CSS Settings
		(function(){

			$DOM.toast.classList.add(PLUGIN_NAME);
			$DOM.toast.classList.add(PLUGIN_NAME+'-opening');
			$DOM.toastCapsule.classList.add(PLUGIN_NAME+'-capsule');
			$DOM.toastBody.classList.add(PLUGIN_NAME + '-body');
			$DOM.toastTexts.classList.add(PLUGIN_NAME + '-texts');

			if(ISMOBILE || window.innerWidth <= MOBILEWIDTH){
				if(settings.transitionInMobile)
					$DOM.toast.classList.add(settings.transitionInMobile);
			} else {
				if(settings.transitionIn)
					$DOM.toast.classList.add(settings.transitionIn);
			}

			if(settings.class){
				var classes = settings.class.split(' ');
				forEach(classes, function (value, index) {
					$DOM.toast.classList.add(value);
				});
			}

			if(settings.id){ $DOM.toast.id = settings.id; }

			if(settings.rtl){
				$DOM.toast.classList.add(PLUGIN_NAME + '-rtl');
				$DOM.toast.setAttribute('dir', 'rtl');
			}

			if(settings.layout > 1){ $DOM.toast.classList.add(PLUGIN_NAME+'-layout'+settings.layout); }

			if(settings.balloon){ $DOM.toast.classList.add(PLUGIN_NAME+'-balloon'); }

			if(settings.maxWidth){
				if( !isNaN(settings.maxWidth) ){
					$DOM.toast.style.maxWidth = settings.maxWidth+'px';
				} else {
					$DOM.toast.style.maxWidth = settings.maxWidth;
				}
			}

			if(settings.theme !== '' || settings.theme !== 'light') {

				$DOM.toast.classList.add(PLUGIN_NAME+'-theme-'+settings.theme);
			}

			if(settings.color) { //#, rgb, rgba, hsl
				
				if( isColor(settings.color) ){
					$DOM.toast.style.background = settings.color;
				} else {
					$DOM.toast.classList.add(PLUGIN_NAME+'-color-'+settings.color);
				}
			}

			if(settings.backgroundColor) {
				$DOM.toast.style.background = settings.backgroundColor;
				if(settings.balloon){
					$DOM.toast.style.borderColor = settings.backgroundColor;				
				}
			}
		})();

		// Cover image
		(function(){
			if(settings.image) {
				$DOM.cover.classList.add(PLUGIN_NAME + '-cover');
				$DOM.cover.style.width = settings.imageWidth + 'px';

				if(isBase64(settings.image.replace(/ /g,''))){
					$DOM.cover.style.backgroundImage = 'url(data:image/png;base64,' + settings.image.replace(/ /g,'') + ')';
				} else {
					$DOM.cover.style.backgroundImage = 'url(' + settings.image + ')';
				}

				if(settings.rtl){
					$DOM.toastBody.style.marginRight = (settings.imageWidth + 10) + 'px';
				} else {
					$DOM.toastBody.style.marginLeft = (settings.imageWidth + 10) + 'px';				
				}
				$DOM.toast.appendChild($DOM.cover);
			}
		})();

		// Button close
		(function(){
			if(settings.close){
				
				$DOM.buttonClose = document.createElement('button');
				$DOM.buttonClose.type = 'button';
				$DOM.buttonClose.classList.add(PLUGIN_NAME + '-close');
				$DOM.buttonClose.addEventListener('click', function (e) {
					var button = e.target;
					that.hide(settings, $DOM.toast, 'button');
				});
				$DOM.toast.appendChild($DOM.buttonClose);
			} else {
				if(settings.rtl){
					$DOM.toast.style.paddingLeft = '18px';
				} else {
					$DOM.toast.style.paddingRight = '18px';
				}
			}
		})();

		// Progress Bar & Timeout
		(function(){

			if(settings.progressBar){
				$DOM.progressBar = document.createElement('div');
				$DOM.progressBarDiv = document.createElement('div');
				$DOM.progressBar.classList.add(PLUGIN_NAME + '-progressbar');
				$DOM.progressBarDiv.style.background = settings.progressBarColor;
				$DOM.progressBar.appendChild($DOM.progressBarDiv);
				$DOM.toast.appendChild($DOM.progressBar);
			}

			if(settings.timeout) {

				if(settings.pauseOnHover && !settings.resetOnHover){
					
					$DOM.toast.addEventListener('mouseenter', function (e) {
						that.progress(settings, $DOM.toast).pause();
					});
					$DOM.toast.addEventListener('mouseleave', function (e) {
						that.progress(settings, $DOM.toast).resume();
					});
				}

				if(settings.resetOnHover){

					$DOM.toast.addEventListener('mouseenter', function (e) {
						that.progress(settings, $DOM.toast).reset();
					});
					$DOM.toast.addEventListener('mouseleave', function (e) {
						that.progress(settings, $DOM.toast).start();
					});
				}
			}
		})();

		// Icon
		(function(){

			if(settings.iconUrl) {

				$DOM.icon.setAttribute('class', PLUGIN_NAME + '-icon');
				$DOM.icon.setAttribute('src', settings.iconUrl);

			} else if(settings.icon) {
				$DOM.icon.setAttribute('class', PLUGIN_NAME + '-icon ' + settings.icon);
				
				if(settings.iconText){
					$DOM.icon.appendChild(document.createTextNode(settings.iconText));
				}
				
				if(settings.iconColor){
					$DOM.icon.style.color = settings.iconColor;
				}				
			}

			if(settings.icon || settings.iconUrl) {

				if(settings.rtl){
					$DOM.toastBody.style.paddingRight = '33px';
				} else {
					$DOM.toastBody.style.paddingLeft = '33px';				
				}

				$DOM.toastBody.appendChild($DOM.icon);
			}

		})();

		// Title & Message
		(function(){
			if(settings.title.length > 0) {

				$DOM.strong = document.createElement('strong');
				$DOM.strong.classList.add(PLUGIN_NAME + '-title');
				$DOM.strong.appendChild(createFragElem(settings.title));
				$DOM.toastTexts.appendChild($DOM.strong);

				if(settings.titleColor) {
					$DOM.strong.style.color = settings.titleColor;
				}
				if(settings.titleSize) {
					if( !isNaN(settings.titleSize) ){
						$DOM.strong.style.fontSize = settings.titleSize+'px';
					} else {
						$DOM.strong.style.fontSize = settings.titleSize;
					}
				}
				if(settings.titleLineHeight) {
					if( !isNaN(settings.titleSize) ){
						$DOM.strong.style.lineHeight = settings.titleLineHeight+'px';
					} else {
						$DOM.strong.style.lineHeight = settings.titleLineHeight;
					}
				}
			}

			if(settings.message.length > 0) {

				$DOM.p = document.createElement('p');
				$DOM.p.classList.add(PLUGIN_NAME + '-message');
				$DOM.p.appendChild(createFragElem(settings.message));
				$DOM.toastTexts.appendChild($DOM.p);

				if(settings.messageColor) {
					$DOM.p.style.color = settings.messageColor;
				}
				if(settings.messageSize) {
					if( !isNaN(settings.titleSize) ){
						$DOM.p.style.fontSize = settings.messageSize+'px';
					} else {
						$DOM.p.style.fontSize = settings.messageSize;
					}
				}
				if(settings.messageLineHeight) {
					
					if( !isNaN(settings.titleSize) ){
						$DOM.p.style.lineHeight = settings.messageLineHeight+'px';
					} else {
						$DOM.p.style.lineHeight = settings.messageLineHeight;
					}
				}
			}

			if(settings.title.length > 0 && settings.message.length > 0) {
				if(settings.rtl){
					$DOM.strong.style.marginLeft = '10px';
				} else if(settings.layout !== 2 && !settings.rtl) {
					$DOM.strong.style.marginRight = '10px';	
				}
			}
		})();

		$DOM.toastBody.appendChild($DOM.toastTexts);

		// Inputs
		var $inputs;
		(function(){
			if(settings.inputs.length > 0) {

				$DOM.inputs.classList.add(PLUGIN_NAME + '-inputs');

				forEach(settings.inputs, function (value, index) {
					$DOM.inputs.appendChild(createFragElem(value[0]));

					$inputs = $DOM.inputs.childNodes;

					$inputs[index].classList.add(PLUGIN_NAME + '-inputs-child');

					if(value[3]){
						setTimeout(function() {
							$inputs[index].focus();
						}, 300);
					}

					$inputs[index].addEventListener(value[1], function (e) {
						var ts = value[2];
						return ts(that, $DOM.toast, this, e);
					});
				});
				$DOM.toastBody.appendChild($DOM.inputs);
			}
		})();

		// Buttons
		(function(){
			if(settings.buttons.length > 0) {

				$DOM.buttons.classList.add(PLUGIN_NAME + '-buttons');

				forEach(settings.buttons, function (value, index) {
					$DOM.buttons.appendChild(createFragElem(value[0]));

					var $btns = $DOM.buttons.childNodes;

					$btns[index].classList.add(PLUGIN_NAME + '-buttons-child');

					if(value[2]){
						setTimeout(function() {
							$btns[index].focus();
						}, 300);
					}

					$btns[index].addEventListener('click', function (e) {
						e.preventDefault();
						var ts = value[1];
						return ts(that, $DOM.toast, this, e, $inputs);
					});
				});
			}
			$DOM.toastBody.appendChild($DOM.buttons);
		})();

		if(settings.message.length > 0 && (settings.inputs.length > 0 || settings.buttons.length > 0)) {
			$DOM.p.style.marginBottom = '0';
		}

		if(settings.inputs.length > 0 || settings.buttons.length > 0){
			if(settings.rtl){
				$DOM.toastTexts.style.marginLeft = '10px';
			} else {
				$DOM.toastTexts.style.marginRight = '10px';
			}
			if(settings.inputs.length > 0 && settings.buttons.length > 0){
				if(settings.rtl){
					$DOM.inputs.style.marginLeft = '8px';
				} else {
					$DOM.inputs.style.marginRight = '8px';
				}
			}
		}

		// Wrap
		(function(){
			$DOM.toastCapsule.style.visibility = 'hidden';
			setTimeout(function() {
				var H = $DOM.toast.offsetHeight;
				var style = $DOM.toast.currentStyle || window.getComputedStyle($DOM.toast);
				var marginTop = style.marginTop;
					marginTop = marginTop.split('px');
					marginTop = parseInt(marginTop[0]);
				var marginBottom = style.marginBottom;
					marginBottom = marginBottom.split('px');
					marginBottom = parseInt(marginBottom[0]);

				$DOM.toastCapsule.style.visibility = '';
				$DOM.toastCapsule.style.height = (H+marginBottom+marginTop)+'px';

				setTimeout(function() {
					$DOM.toastCapsule.style.height = 'auto';
					if(settings.target){
						$DOM.toastCapsule.style.overflow = 'visible';
					}
				}, 500);

				if(settings.timeout) {
					that.progress(settings, $DOM.toast).start();
				}
			}, 100);
		})();

		// Target
		(function(){
			var position = settings.position;

			if(settings.target){

				$DOM.wrapper = document.querySelector(settings.target);
				$DOM.wrapper.classList.add(PLUGIN_NAME + '-target');

				if(settings.targetFirst) {
					$DOM.wrapper.insertBefore($DOM.toastCapsule, $DOM.wrapper.firstChild);
				} else {
					$DOM.wrapper.appendChild($DOM.toastCapsule);
				}

			} else {

				if( POSITIONS.indexOf(settings.position) == -1 ){
					console.warn('['+PLUGIN_NAME+'] Incorrect position.\nIt can be › ' + POSITIONS);
					return;
				}

				if(ISMOBILE || window.innerWidth <= MOBILEWIDTH){
					if(settings.position == 'bottomLeft' || settings.position == 'bottomRight' || settings.position == 'bottomCenter'){
						position = PLUGIN_NAME+'-wrapper-bottomCenter';
					}
					else if(settings.position == 'topLeft' || settings.position == 'topRight' || settings.position == 'topCenter'){
						position = PLUGIN_NAME+'-wrapper-topCenter';
					}
					else {
						position = PLUGIN_NAME+'-wrapper-center';
					}
				} else {
					position = PLUGIN_NAME+'-wrapper-'+position;
				}
				$DOM.wrapper = document.querySelector('.' + PLUGIN_NAME + '-wrapper.'+position);

				if(!$DOM.wrapper) {
					$DOM.wrapper = document.createElement('div');
					$DOM.wrapper.classList.add(PLUGIN_NAME + '-wrapper');
					$DOM.wrapper.classList.add(position);
					document.body.appendChild($DOM.wrapper);
				}
				if(settings.position == 'topLeft' || settings.position == 'topCenter' || settings.position == 'topRight'){
					$DOM.wrapper.insertBefore($DOM.toastCapsule, $DOM.wrapper.firstChild);
				} else {
					$DOM.wrapper.appendChild($DOM.toastCapsule);
				}
			}

			if(!isNaN(settings.zindex)) {
				$DOM.wrapper.style.zIndex = settings.zindex;
			} else {
				console.warn('['+PLUGIN_NAME+'] Invalid zIndex.');
			}
		})();

		// Overlay
		(function(){

			if(settings.overlay) {

				if( document.querySelector('.'+PLUGIN_NAME+'-overlay.fadeIn') !== null ){

					$DOM.overlay = document.querySelector('.'+PLUGIN_NAME+'-overlay');
					$DOM.overlay.setAttribute('data-iziToast-ref', $DOM.overlay.getAttribute('data-iziToast-ref') + ',' + settings.ref);

					if(!isNaN(settings.zindex) && settings.zindex !== null) {
						$DOM.overlay.style.zIndex = settings.zindex-1;
					}

				} else {

					$DOM.overlay.classList.add(PLUGIN_NAME+'-overlay');
					$DOM.overlay.classList.add('fadeIn');
					$DOM.overlay.style.background = settings.overlayColor;
					$DOM.overlay.setAttribute('data-iziToast-ref', settings.ref);
					if(!isNaN(settings.zindex) && settings.zindex !== null) {
						$DOM.overlay.style.zIndex = settings.zindex-1;
					}
					document.querySelector('body').appendChild($DOM.overlay);
				}

				if(settings.overlayClose) {

					$DOM.overlay.removeEventListener('click', {});
					$DOM.overlay.addEventListener('click', function (e) {
						that.hide(settings, $DOM.toast, 'overlay');
					});
				} else {
					$DOM.overlay.removeEventListener('click', {});
				}
			}			
		})();

		// Inside animations
		(function(){
			if(settings.animateInside){
				$DOM.toast.classList.add(PLUGIN_NAME+'-animateInside');
			
				var animationTimes = [200, 100, 300];
				if(settings.transitionIn == 'bounceInLeft' || settings.transitionIn == 'bounceInRight'){
					animationTimes = [400, 200, 400];
				}

				if(settings.title.length > 0) {
					setTimeout(function(){
						$DOM.strong.classList.add('slideIn');
					}, animationTimes[0]);
				}

				if(settings.message.length > 0) {
					setTimeout(function(){
						$DOM.p.classList.add('slideIn');
					}, animationTimes[1]);
				}

				if(settings.icon || settings.iconUrl) {
					setTimeout(function(){
						$DOM.icon.classList.add('revealIn');
					}, animationTimes[2]);
				}

				var counter = 150;
				if(settings.buttons.length > 0 && $DOM.buttons) {

					setTimeout(function(){

						forEach($DOM.buttons.childNodes, function(element, index) {

							setTimeout(function(){
								element.classList.add('revealIn');
							}, counter);
							counter = counter + 150;
						});

					}, settings.inputs.length > 0 ? 150 : 0);
				}

				if(settings.inputs.length > 0 && $DOM.inputs) {
					counter = 150;
					forEach($DOM.inputs.childNodes, function(element, index) {

						setTimeout(function(){
							element.classList.add('revealIn');
						}, counter);
						counter = counter + 150;
					});
				}
			}
		})();

		settings.onOpening.apply(null, [settings, $DOM.toast]);

		try {
			var event = new CustomEvent(PLUGIN_NAME + '-opening', {detail: settings, bubbles: true, cancelable: true});
			document.dispatchEvent(event);
		} catch(ex){
			console.warn(ex);
		}

		setTimeout(function() {

			$DOM.toast.classList.remove(PLUGIN_NAME+'-opening');
			$DOM.toast.classList.add(PLUGIN_NAME+'-opened');

			try {
				var event = new CustomEvent(PLUGIN_NAME + '-opened', {detail: settings, bubbles: true, cancelable: true});
				document.dispatchEvent(event);
			} catch(ex){
				console.warn(ex);
			}

			settings.onOpened.apply(null, [settings, $DOM.toast]);
		}, 1000);

		if(settings.drag){

			if(ACCEPTSTOUCH) {

			    $DOM.toast.addEventListener('touchstart', function(e) {
			        drag.startMoving(this, that, settings, e);
			    }, false);

			    $DOM.toast.addEventListener('touchend', function(e) {
			        drag.stopMoving(this, e);
			    }, false);
			} else {

			    $DOM.toast.addEventListener('mousedown', function(e) {
			    	e.preventDefault();
			        drag.startMoving(this, that, settings, e);
			    }, false);

			    $DOM.toast.addEventListener('mouseup', function(e) {
			    	e.preventDefault();
			        drag.stopMoving(this, e);
			    }, false);
			}
		}

		if(settings.closeOnEscape) {

			document.addEventListener('keyup', function (evt) {
				evt = evt || window.event;
				if(evt.keyCode == 27) {
				    that.hide(settings, $DOM.toast, 'esc');
				}
			});
		}

		if(settings.closeOnClick) {
			$DOM.toast.addEventListener('click', function (evt) {
				that.hide(settings, $DOM.toast, 'toast');
			});
		}

		that.toast = $DOM.toast;		
	};
	

	return $iziToast;
});

/***/ }),

/***/ "./node_modules/process/browser.js":
/*!*****************************************!*\
  !*** ./node_modules/process/browser.js ***!
  \*****************************************/
/***/ ((module) => {

// shim for using process in browser
var process = module.exports = {};

// cached from whatever global is present so that test runners that stub it
// don't break things.  But we need to wrap it in a try catch in case it is
// wrapped in strict mode code which doesn't define any globals.  It's inside a
// function because try/catches deoptimize in certain engines.

var cachedSetTimeout;
var cachedClearTimeout;

function defaultSetTimout() {
    throw new Error('setTimeout has not been defined');
}
function defaultClearTimeout () {
    throw new Error('clearTimeout has not been defined');
}
(function () {
    try {
        if (typeof setTimeout === 'function') {
            cachedSetTimeout = setTimeout;
        } else {
            cachedSetTimeout = defaultSetTimout;
        }
    } catch (e) {
        cachedSetTimeout = defaultSetTimout;
    }
    try {
        if (typeof clearTimeout === 'function') {
            cachedClearTimeout = clearTimeout;
        } else {
            cachedClearTimeout = defaultClearTimeout;
        }
    } catch (e) {
        cachedClearTimeout = defaultClearTimeout;
    }
} ())
function runTimeout(fun) {
    if (cachedSetTimeout === setTimeout) {
        //normal enviroments in sane situations
        return setTimeout(fun, 0);
    }
    // if setTimeout wasn't available but was latter defined
    if ((cachedSetTimeout === defaultSetTimout || !cachedSetTimeout) && setTimeout) {
        cachedSetTimeout = setTimeout;
        return setTimeout(fun, 0);
    }
    try {
        // when when somebody has screwed with setTimeout but no I.E. maddness
        return cachedSetTimeout(fun, 0);
    } catch(e){
        try {
            // When we are in I.E. but the script has been evaled so I.E. doesn't trust the global object when called normally
            return cachedSetTimeout.call(null, fun, 0);
        } catch(e){
            // same as above but when it's a version of I.E. that must have the global object for 'this', hopfully our context correct otherwise it will throw a global error
            return cachedSetTimeout.call(this, fun, 0);
        }
    }


}
function runClearTimeout(marker) {
    if (cachedClearTimeout === clearTimeout) {
        //normal enviroments in sane situations
        return clearTimeout(marker);
    }
    // if clearTimeout wasn't available but was latter defined
    if ((cachedClearTimeout === defaultClearTimeout || !cachedClearTimeout) && clearTimeout) {
        cachedClearTimeout = clearTimeout;
        return clearTimeout(marker);
    }
    try {
        // when when somebody has screwed with setTimeout but no I.E. maddness
        return cachedClearTimeout(marker);
    } catch (e){
        try {
            // When we are in I.E. but the script has been evaled so I.E. doesn't  trust the global object when called normally
            return cachedClearTimeout.call(null, marker);
        } catch (e){
            // same as above but when it's a version of I.E. that must have the global object for 'this', hopfully our context correct otherwise it will throw a global error.
            // Some versions of I.E. have different rules for clearTimeout vs setTimeout
            return cachedClearTimeout.call(this, marker);
        }
    }



}
var queue = [];
var draining = false;
var currentQueue;
var queueIndex = -1;

function cleanUpNextTick() {
    if (!draining || !currentQueue) {
        return;
    }
    draining = false;
    if (currentQueue.length) {
        queue = currentQueue.concat(queue);
    } else {
        queueIndex = -1;
    }
    if (queue.length) {
        drainQueue();
    }
}

function drainQueue() {
    if (draining) {
        return;
    }
    var timeout = runTimeout(cleanUpNextTick);
    draining = true;

    var len = queue.length;
    while(len) {
        currentQueue = queue;
        queue = [];
        while (++queueIndex < len) {
            if (currentQueue) {
                currentQueue[queueIndex].run();
            }
        }
        queueIndex = -1;
        len = queue.length;
    }
    currentQueue = null;
    draining = false;
    runClearTimeout(timeout);
}

process.nextTick = function (fun) {
    var args = new Array(arguments.length - 1);
    if (arguments.length > 1) {
        for (var i = 1; i < arguments.length; i++) {
            args[i - 1] = arguments[i];
        }
    }
    queue.push(new Item(fun, args));
    if (queue.length === 1 && !draining) {
        runTimeout(drainQueue);
    }
};

// v8 likes predictible objects
function Item(fun, array) {
    this.fun = fun;
    this.array = array;
}
Item.prototype.run = function () {
    this.fun.apply(null, this.array);
};
process.title = 'browser';
process.browser = true;
process.env = {};
process.argv = [];
process.version = ''; // empty string to avoid regexp issues
process.versions = {};

function noop() {}

process.on = noop;
process.addListener = noop;
process.once = noop;
process.off = noop;
process.removeListener = noop;
process.removeAllListeners = noop;
process.emit = noop;
process.prependListener = noop;
process.prependOnceListener = noop;

process.listeners = function (name) { return [] }

process.binding = function (name) {
    throw new Error('process.binding is not supported');
};

process.cwd = function () { return '/' };
process.chdir = function (dir) {
    throw new Error('process.chdir is not supported');
};
process.umask = function() { return 0; };


/***/ }),

/***/ "./node_modules/regenerator-runtime/runtime.js":
/*!*****************************************************!*\
  !*** ./node_modules/regenerator-runtime/runtime.js ***!
  \*****************************************************/
/***/ ((module) => {

/**
 * Copyright (c) 2014-present, Facebook, Inc.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

var runtime = (function (exports) {
  "use strict";

  var Op = Object.prototype;
  var hasOwn = Op.hasOwnProperty;
  var undefined; // More compressible than void 0.
  var $Symbol = typeof Symbol === "function" ? Symbol : {};
  var iteratorSymbol = $Symbol.iterator || "@@iterator";
  var asyncIteratorSymbol = $Symbol.asyncIterator || "@@asyncIterator";
  var toStringTagSymbol = $Symbol.toStringTag || "@@toStringTag";

  function define(obj, key, value) {
    Object.defineProperty(obj, key, {
      value: value,
      enumerable: true,
      configurable: true,
      writable: true
    });
    return obj[key];
  }
  try {
    // IE 8 has a broken Object.defineProperty that only works on DOM objects.
    define({}, "");
  } catch (err) {
    define = function(obj, key, value) {
      return obj[key] = value;
    };
  }

  function wrap(innerFn, outerFn, self, tryLocsList) {
    // If outerFn provided and outerFn.prototype is a Generator, then outerFn.prototype instanceof Generator.
    var protoGenerator = outerFn && outerFn.prototype instanceof Generator ? outerFn : Generator;
    var generator = Object.create(protoGenerator.prototype);
    var context = new Context(tryLocsList || []);

    // The ._invoke method unifies the implementations of the .next,
    // .throw, and .return methods.
    generator._invoke = makeInvokeMethod(innerFn, self, context);

    return generator;
  }
  exports.wrap = wrap;

  // Try/catch helper to minimize deoptimizations. Returns a completion
  // record like context.tryEntries[i].completion. This interface could
  // have been (and was previously) designed to take a closure to be
  // invoked without arguments, but in all the cases we care about we
  // already have an existing method we want to call, so there's no need
  // to create a new function object. We can even get away with assuming
  // the method takes exactly one argument, since that happens to be true
  // in every case, so we don't have to touch the arguments object. The
  // only additional allocation required is the completion record, which
  // has a stable shape and so hopefully should be cheap to allocate.
  function tryCatch(fn, obj, arg) {
    try {
      return { type: "normal", arg: fn.call(obj, arg) };
    } catch (err) {
      return { type: "throw", arg: err };
    }
  }

  var GenStateSuspendedStart = "suspendedStart";
  var GenStateSuspendedYield = "suspendedYield";
  var GenStateExecuting = "executing";
  var GenStateCompleted = "completed";

  // Returning this object from the innerFn has the same effect as
  // breaking out of the dispatch switch statement.
  var ContinueSentinel = {};

  // Dummy constructor functions that we use as the .constructor and
  // .constructor.prototype properties for functions that return Generator
  // objects. For full spec compliance, you may wish to configure your
  // minifier not to mangle the names of these two functions.
  function Generator() {}
  function GeneratorFunction() {}
  function GeneratorFunctionPrototype() {}

  // This is a polyfill for %IteratorPrototype% for environments that
  // don't natively support it.
  var IteratorPrototype = {};
  IteratorPrototype[iteratorSymbol] = function () {
    return this;
  };

  var getProto = Object.getPrototypeOf;
  var NativeIteratorPrototype = getProto && getProto(getProto(values([])));
  if (NativeIteratorPrototype &&
      NativeIteratorPrototype !== Op &&
      hasOwn.call(NativeIteratorPrototype, iteratorSymbol)) {
    // This environment has a native %IteratorPrototype%; use it instead
    // of the polyfill.
    IteratorPrototype = NativeIteratorPrototype;
  }

  var Gp = GeneratorFunctionPrototype.prototype =
    Generator.prototype = Object.create(IteratorPrototype);
  GeneratorFunction.prototype = Gp.constructor = GeneratorFunctionPrototype;
  GeneratorFunctionPrototype.constructor = GeneratorFunction;
  GeneratorFunction.displayName = define(
    GeneratorFunctionPrototype,
    toStringTagSymbol,
    "GeneratorFunction"
  );

  // Helper for defining the .next, .throw, and .return methods of the
  // Iterator interface in terms of a single ._invoke method.
  function defineIteratorMethods(prototype) {
    ["next", "throw", "return"].forEach(function(method) {
      define(prototype, method, function(arg) {
        return this._invoke(method, arg);
      });
    });
  }

  exports.isGeneratorFunction = function(genFun) {
    var ctor = typeof genFun === "function" && genFun.constructor;
    return ctor
      ? ctor === GeneratorFunction ||
        // For the native GeneratorFunction constructor, the best we can
        // do is to check its .name property.
        (ctor.displayName || ctor.name) === "GeneratorFunction"
      : false;
  };

  exports.mark = function(genFun) {
    if (Object.setPrototypeOf) {
      Object.setPrototypeOf(genFun, GeneratorFunctionPrototype);
    } else {
      genFun.__proto__ = GeneratorFunctionPrototype;
      define(genFun, toStringTagSymbol, "GeneratorFunction");
    }
    genFun.prototype = Object.create(Gp);
    return genFun;
  };

  // Within the body of any async function, `await x` is transformed to
  // `yield regeneratorRuntime.awrap(x)`, so that the runtime can test
  // `hasOwn.call(value, "__await")` to determine if the yielded value is
  // meant to be awaited.
  exports.awrap = function(arg) {
    return { __await: arg };
  };

  function AsyncIterator(generator, PromiseImpl) {
    function invoke(method, arg, resolve, reject) {
      var record = tryCatch(generator[method], generator, arg);
      if (record.type === "throw") {
        reject(record.arg);
      } else {
        var result = record.arg;
        var value = result.value;
        if (value &&
            typeof value === "object" &&
            hasOwn.call(value, "__await")) {
          return PromiseImpl.resolve(value.__await).then(function(value) {
            invoke("next", value, resolve, reject);
          }, function(err) {
            invoke("throw", err, resolve, reject);
          });
        }

        return PromiseImpl.resolve(value).then(function(unwrapped) {
          // When a yielded Promise is resolved, its final value becomes
          // the .value of the Promise<{value,done}> result for the
          // current iteration.
          result.value = unwrapped;
          resolve(result);
        }, function(error) {
          // If a rejected Promise was yielded, throw the rejection back
          // into the async generator function so it can be handled there.
          return invoke("throw", error, resolve, reject);
        });
      }
    }

    var previousPromise;

    function enqueue(method, arg) {
      function callInvokeWithMethodAndArg() {
        return new PromiseImpl(function(resolve, reject) {
          invoke(method, arg, resolve, reject);
        });
      }

      return previousPromise =
        // If enqueue has been called before, then we want to wait until
        // all previous Promises have been resolved before calling invoke,
        // so that results are always delivered in the correct order. If
        // enqueue has not been called before, then it is important to
        // call invoke immediately, without waiting on a callback to fire,
        // so that the async generator function has the opportunity to do
        // any necessary setup in a predictable way. This predictability
        // is why the Promise constructor synchronously invokes its
        // executor callback, and why async functions synchronously
        // execute code before the first await. Since we implement simple
        // async functions in terms of async generators, it is especially
        // important to get this right, even though it requires care.
        previousPromise ? previousPromise.then(
          callInvokeWithMethodAndArg,
          // Avoid propagating failures to Promises returned by later
          // invocations of the iterator.
          callInvokeWithMethodAndArg
        ) : callInvokeWithMethodAndArg();
    }

    // Define the unified helper method that is used to implement .next,
    // .throw, and .return (see defineIteratorMethods).
    this._invoke = enqueue;
  }

  defineIteratorMethods(AsyncIterator.prototype);
  AsyncIterator.prototype[asyncIteratorSymbol] = function () {
    return this;
  };
  exports.AsyncIterator = AsyncIterator;

  // Note that simple async functions are implemented on top of
  // AsyncIterator objects; they just return a Promise for the value of
  // the final result produced by the iterator.
  exports.async = function(innerFn, outerFn, self, tryLocsList, PromiseImpl) {
    if (PromiseImpl === void 0) PromiseImpl = Promise;

    var iter = new AsyncIterator(
      wrap(innerFn, outerFn, self, tryLocsList),
      PromiseImpl
    );

    return exports.isGeneratorFunction(outerFn)
      ? iter // If outerFn is a generator, return the full iterator.
      : iter.next().then(function(result) {
          return result.done ? result.value : iter.next();
        });
  };

  function makeInvokeMethod(innerFn, self, context) {
    var state = GenStateSuspendedStart;

    return function invoke(method, arg) {
      if (state === GenStateExecuting) {
        throw new Error("Generator is already running");
      }

      if (state === GenStateCompleted) {
        if (method === "throw") {
          throw arg;
        }

        // Be forgiving, per 25.3.3.3.3 of the spec:
        // https://people.mozilla.org/~jorendorff/es6-draft.html#sec-generatorresume
        return doneResult();
      }

      context.method = method;
      context.arg = arg;

      while (true) {
        var delegate = context.delegate;
        if (delegate) {
          var delegateResult = maybeInvokeDelegate(delegate, context);
          if (delegateResult) {
            if (delegateResult === ContinueSentinel) continue;
            return delegateResult;
          }
        }

        if (context.method === "next") {
          // Setting context._sent for legacy support of Babel's
          // function.sent implementation.
          context.sent = context._sent = context.arg;

        } else if (context.method === "throw") {
          if (state === GenStateSuspendedStart) {
            state = GenStateCompleted;
            throw context.arg;
          }

          context.dispatchException(context.arg);

        } else if (context.method === "return") {
          context.abrupt("return", context.arg);
        }

        state = GenStateExecuting;

        var record = tryCatch(innerFn, self, context);
        if (record.type === "normal") {
          // If an exception is thrown from innerFn, we leave state ===
          // GenStateExecuting and loop back for another invocation.
          state = context.done
            ? GenStateCompleted
            : GenStateSuspendedYield;

          if (record.arg === ContinueSentinel) {
            continue;
          }

          return {
            value: record.arg,
            done: context.done
          };

        } else if (record.type === "throw") {
          state = GenStateCompleted;
          // Dispatch the exception by looping back around to the
          // context.dispatchException(context.arg) call above.
          context.method = "throw";
          context.arg = record.arg;
        }
      }
    };
  }

  // Call delegate.iterator[context.method](context.arg) and handle the
  // result, either by returning a { value, done } result from the
  // delegate iterator, or by modifying context.method and context.arg,
  // setting context.delegate to null, and returning the ContinueSentinel.
  function maybeInvokeDelegate(delegate, context) {
    var method = delegate.iterator[context.method];
    if (method === undefined) {
      // A .throw or .return when the delegate iterator has no .throw
      // method always terminates the yield* loop.
      context.delegate = null;

      if (context.method === "throw") {
        // Note: ["return"] must be used for ES3 parsing compatibility.
        if (delegate.iterator["return"]) {
          // If the delegate iterator has a return method, give it a
          // chance to clean up.
          context.method = "return";
          context.arg = undefined;
          maybeInvokeDelegate(delegate, context);

          if (context.method === "throw") {
            // If maybeInvokeDelegate(context) changed context.method from
            // "return" to "throw", let that override the TypeError below.
            return ContinueSentinel;
          }
        }

        context.method = "throw";
        context.arg = new TypeError(
          "The iterator does not provide a 'throw' method");
      }

      return ContinueSentinel;
    }

    var record = tryCatch(method, delegate.iterator, context.arg);

    if (record.type === "throw") {
      context.method = "throw";
      context.arg = record.arg;
      context.delegate = null;
      return ContinueSentinel;
    }

    var info = record.arg;

    if (! info) {
      context.method = "throw";
      context.arg = new TypeError("iterator result is not an object");
      context.delegate = null;
      return ContinueSentinel;
    }

    if (info.done) {
      // Assign the result of the finished delegate to the temporary
      // variable specified by delegate.resultName (see delegateYield).
      context[delegate.resultName] = info.value;

      // Resume execution at the desired location (see delegateYield).
      context.next = delegate.nextLoc;

      // If context.method was "throw" but the delegate handled the
      // exception, let the outer generator proceed normally. If
      // context.method was "next", forget context.arg since it has been
      // "consumed" by the delegate iterator. If context.method was
      // "return", allow the original .return call to continue in the
      // outer generator.
      if (context.method !== "return") {
        context.method = "next";
        context.arg = undefined;
      }

    } else {
      // Re-yield the result returned by the delegate method.
      return info;
    }

    // The delegate iterator is finished, so forget it and continue with
    // the outer generator.
    context.delegate = null;
    return ContinueSentinel;
  }

  // Define Generator.prototype.{next,throw,return} in terms of the
  // unified ._invoke helper method.
  defineIteratorMethods(Gp);

  define(Gp, toStringTagSymbol, "Generator");

  // A Generator should always return itself as the iterator object when the
  // @@iterator function is called on it. Some browsers' implementations of the
  // iterator prototype chain incorrectly implement this, causing the Generator
  // object to not be returned from this call. This ensures that doesn't happen.
  // See https://github.com/facebook/regenerator/issues/274 for more details.
  Gp[iteratorSymbol] = function() {
    return this;
  };

  Gp.toString = function() {
    return "[object Generator]";
  };

  function pushTryEntry(locs) {
    var entry = { tryLoc: locs[0] };

    if (1 in locs) {
      entry.catchLoc = locs[1];
    }

    if (2 in locs) {
      entry.finallyLoc = locs[2];
      entry.afterLoc = locs[3];
    }

    this.tryEntries.push(entry);
  }

  function resetTryEntry(entry) {
    var record = entry.completion || {};
    record.type = "normal";
    delete record.arg;
    entry.completion = record;
  }

  function Context(tryLocsList) {
    // The root entry object (effectively a try statement without a catch
    // or a finally block) gives us a place to store values thrown from
    // locations where there is no enclosing try statement.
    this.tryEntries = [{ tryLoc: "root" }];
    tryLocsList.forEach(pushTryEntry, this);
    this.reset(true);
  }

  exports.keys = function(object) {
    var keys = [];
    for (var key in object) {
      keys.push(key);
    }
    keys.reverse();

    // Rather than returning an object with a next method, we keep
    // things simple and return the next function itself.
    return function next() {
      while (keys.length) {
        var key = keys.pop();
        if (key in object) {
          next.value = key;
          next.done = false;
          return next;
        }
      }

      // To avoid creating an additional object, we just hang the .value
      // and .done properties off the next function object itself. This
      // also ensures that the minifier will not anonymize the function.
      next.done = true;
      return next;
    };
  };

  function values(iterable) {
    if (iterable) {
      var iteratorMethod = iterable[iteratorSymbol];
      if (iteratorMethod) {
        return iteratorMethod.call(iterable);
      }

      if (typeof iterable.next === "function") {
        return iterable;
      }

      if (!isNaN(iterable.length)) {
        var i = -1, next = function next() {
          while (++i < iterable.length) {
            if (hasOwn.call(iterable, i)) {
              next.value = iterable[i];
              next.done = false;
              return next;
            }
          }

          next.value = undefined;
          next.done = true;

          return next;
        };

        return next.next = next;
      }
    }

    // Return an iterator with no values.
    return { next: doneResult };
  }
  exports.values = values;

  function doneResult() {
    return { value: undefined, done: true };
  }

  Context.prototype = {
    constructor: Context,

    reset: function(skipTempReset) {
      this.prev = 0;
      this.next = 0;
      // Resetting context._sent for legacy support of Babel's
      // function.sent implementation.
      this.sent = this._sent = undefined;
      this.done = false;
      this.delegate = null;

      this.method = "next";
      this.arg = undefined;

      this.tryEntries.forEach(resetTryEntry);

      if (!skipTempReset) {
        for (var name in this) {
          // Not sure about the optimal order of these conditions:
          if (name.charAt(0) === "t" &&
              hasOwn.call(this, name) &&
              !isNaN(+name.slice(1))) {
            this[name] = undefined;
          }
        }
      }
    },

    stop: function() {
      this.done = true;

      var rootEntry = this.tryEntries[0];
      var rootRecord = rootEntry.completion;
      if (rootRecord.type === "throw") {
        throw rootRecord.arg;
      }

      return this.rval;
    },

    dispatchException: function(exception) {
      if (this.done) {
        throw exception;
      }

      var context = this;
      function handle(loc, caught) {
        record.type = "throw";
        record.arg = exception;
        context.next = loc;

        if (caught) {
          // If the dispatched exception was caught by a catch block,
          // then let that catch block handle the exception normally.
          context.method = "next";
          context.arg = undefined;
        }

        return !! caught;
      }

      for (var i = this.tryEntries.length - 1; i >= 0; --i) {
        var entry = this.tryEntries[i];
        var record = entry.completion;

        if (entry.tryLoc === "root") {
          // Exception thrown outside of any try block that could handle
          // it, so set the completion value of the entire function to
          // throw the exception.
          return handle("end");
        }

        if (entry.tryLoc <= this.prev) {
          var hasCatch = hasOwn.call(entry, "catchLoc");
          var hasFinally = hasOwn.call(entry, "finallyLoc");

          if (hasCatch && hasFinally) {
            if (this.prev < entry.catchLoc) {
              return handle(entry.catchLoc, true);
            } else if (this.prev < entry.finallyLoc) {
              return handle(entry.finallyLoc);
            }

          } else if (hasCatch) {
            if (this.prev < entry.catchLoc) {
              return handle(entry.catchLoc, true);
            }

          } else if (hasFinally) {
            if (this.prev < entry.finallyLoc) {
              return handle(entry.finallyLoc);
            }

          } else {
            throw new Error("try statement without catch or finally");
          }
        }
      }
    },

    abrupt: function(type, arg) {
      for (var i = this.tryEntries.length - 1; i >= 0; --i) {
        var entry = this.tryEntries[i];
        if (entry.tryLoc <= this.prev &&
            hasOwn.call(entry, "finallyLoc") &&
            this.prev < entry.finallyLoc) {
          var finallyEntry = entry;
          break;
        }
      }

      if (finallyEntry &&
          (type === "break" ||
           type === "continue") &&
          finallyEntry.tryLoc <= arg &&
          arg <= finallyEntry.finallyLoc) {
        // Ignore the finally entry if control is not jumping to a
        // location outside the try/catch block.
        finallyEntry = null;
      }

      var record = finallyEntry ? finallyEntry.completion : {};
      record.type = type;
      record.arg = arg;

      if (finallyEntry) {
        this.method = "next";
        this.next = finallyEntry.finallyLoc;
        return ContinueSentinel;
      }

      return this.complete(record);
    },

    complete: function(record, afterLoc) {
      if (record.type === "throw") {
        throw record.arg;
      }

      if (record.type === "break" ||
          record.type === "continue") {
        this.next = record.arg;
      } else if (record.type === "return") {
        this.rval = this.arg = record.arg;
        this.method = "return";
        this.next = "end";
      } else if (record.type === "normal" && afterLoc) {
        this.next = afterLoc;
      }

      return ContinueSentinel;
    },

    finish: function(finallyLoc) {
      for (var i = this.tryEntries.length - 1; i >= 0; --i) {
        var entry = this.tryEntries[i];
        if (entry.finallyLoc === finallyLoc) {
          this.complete(entry.completion, entry.afterLoc);
          resetTryEntry(entry);
          return ContinueSentinel;
        }
      }
    },

    "catch": function(tryLoc) {
      for (var i = this.tryEntries.length - 1; i >= 0; --i) {
        var entry = this.tryEntries[i];
        if (entry.tryLoc === tryLoc) {
          var record = entry.completion;
          if (record.type === "throw") {
            var thrown = record.arg;
            resetTryEntry(entry);
          }
          return thrown;
        }
      }

      // The context.catch method must only be called with a location
      // argument that corresponds to a known catch block.
      throw new Error("illegal catch attempt");
    },

    delegateYield: function(iterable, resultName, nextLoc) {
      this.delegate = {
        iterator: values(iterable),
        resultName: resultName,
        nextLoc: nextLoc
      };

      if (this.method === "next") {
        // Deliberately forget the last sent value so that we don't
        // accidentally pass it on to the delegate.
        this.arg = undefined;
      }

      return ContinueSentinel;
    }
  };

  // Regardless of whether this script is executing as a CommonJS module
  // or not, return the runtime object so that we can declare the variable
  // regeneratorRuntime in the outer scope, which allows this module to be
  // injected easily by `bin/regenerator --include-runtime script.js`.
  return exports;

}(
  // If this script is executing as a CommonJS module, use module.exports
  // as the regeneratorRuntime namespace. Otherwise create a new empty
  // object. Either way, the resulting object will be used to initialize
  // the regeneratorRuntime variable at the top of this file.
   true ? module.exports : 0
));

try {
  regeneratorRuntime = runtime;
} catch (accidentalStrictMode) {
  // This module should not be running in strict mode, so the above
  // assignment should always work unless something is misconfigured. Just
  // in case runtime.js accidentally runs in strict mode, we can escape
  // strict mode using a global Function call. This could conceivably fail
  // if a Content Security Policy forbids using Function, but in that case
  // the proper solution is to fix the accidental strict mode problem. If
  // you've misconfigured your bundler to force strict mode and applied a
  // CSP to forbid Function, and you're not willing to fix either of those
  // problems, please detail your unique predicament in a GitHub issue.
  Function("r", "regeneratorRuntime = r")(runtime);
}


/***/ }),

/***/ "./node_modules/sweetalert2/dist/sweetalert2.all.js":
/*!**********************************************************!*\
  !*** ./node_modules/sweetalert2/dist/sweetalert2.all.js ***!
  \**********************************************************/
/***/ (function(module) {

/*!
* sweetalert2 v11.3.0
* Released under the MIT License.
*/
(function (global, factory) {
   true ? module.exports = factory() :
  0;
}(this, function () { 'use strict';

  const DismissReason = Object.freeze({
    cancel: 'cancel',
    backdrop: 'backdrop',
    close: 'close',
    esc: 'esc',
    timer: 'timer'
  });

  const consolePrefix = 'SweetAlert2:';
  /**
   * Filter the unique values into a new array
   * @param arr
   */

  const uniqueArray = arr => {
    const result = [];

    for (let i = 0; i < arr.length; i++) {
      if (result.indexOf(arr[i]) === -1) {
        result.push(arr[i]);
      }
    }

    return result;
  };
  /**
   * Capitalize the first letter of a string
   * @param str
   */

  const capitalizeFirstLetter = str => str.charAt(0).toUpperCase() + str.slice(1);
  /**
   * Convert NodeList to Array
   * @param nodeList
   */

  const toArray = nodeList => Array.prototype.slice.call(nodeList);
  /**
   * Standardise console warnings
   * @param message
   */

  const warn = message => {
    console.warn("".concat(consolePrefix, " ").concat(typeof message === 'object' ? message.join(' ') : message));
  };
  /**
   * Standardise console errors
   * @param message
   */

  const error = message => {
    console.error("".concat(consolePrefix, " ").concat(message));
  };
  /**
   * Private global state for `warnOnce`
   * @type {Array}
   * @private
   */

  const previousWarnOnceMessages = [];
  /**
   * Show a console warning, but only if it hasn't already been shown
   * @param message
   */

  const warnOnce = message => {
    if (!previousWarnOnceMessages.includes(message)) {
      previousWarnOnceMessages.push(message);
      warn(message);
    }
  };
  /**
   * Show a one-time console warning about deprecated params/methods
   */

  const warnAboutDeprecation = (deprecatedParam, useInstead) => {
    warnOnce("\"".concat(deprecatedParam, "\" is deprecated and will be removed in the next major release. Please use \"").concat(useInstead, "\" instead."));
  };
  /**
   * If `arg` is a function, call it (with no arguments or context) and return the result.
   * Otherwise, just pass the value through
   * @param arg
   */

  const callIfFunction = arg => typeof arg === 'function' ? arg() : arg;
  const hasToPromiseFn = arg => arg && typeof arg.toPromise === 'function';
  const asPromise = arg => hasToPromiseFn(arg) ? arg.toPromise() : Promise.resolve(arg);
  const isPromise = arg => arg && Promise.resolve(arg) === arg;

  const isJqueryElement = elem => typeof elem === 'object' && elem.jquery;

  const isElement = elem => elem instanceof Element || isJqueryElement(elem);

  const argsToParams = args => {
    const params = {};

    if (typeof args[0] === 'object' && !isElement(args[0])) {
      Object.assign(params, args[0]);
    } else {
      ['title', 'html', 'icon'].forEach((name, index) => {
        const arg = args[index];

        if (typeof arg === 'string' || isElement(arg)) {
          params[name] = arg;
        } else if (arg !== undefined) {
          error("Unexpected type of ".concat(name, "! Expected \"string\" or \"Element\", got ").concat(typeof arg));
        }
      });
    }

    return params;
  };

  const swalPrefix = 'swal2-';
  const prefix = items => {
    const result = {};

    for (const i in items) {
      result[items[i]] = swalPrefix + items[i];
    }

    return result;
  };
  const swalClasses = prefix(['container', 'shown', 'height-auto', 'iosfix', 'popup', 'modal', 'no-backdrop', 'no-transition', 'toast', 'toast-shown', 'show', 'hide', 'close', 'title', 'html-container', 'actions', 'confirm', 'deny', 'cancel', 'default-outline', 'footer', 'icon', 'icon-content', 'image', 'input', 'file', 'range', 'select', 'radio', 'checkbox', 'label', 'textarea', 'inputerror', 'input-label', 'validation-message', 'progress-steps', 'active-progress-step', 'progress-step', 'progress-step-line', 'loader', 'loading', 'styled', 'top', 'top-start', 'top-end', 'top-left', 'top-right', 'center', 'center-start', 'center-end', 'center-left', 'center-right', 'bottom', 'bottom-start', 'bottom-end', 'bottom-left', 'bottom-right', 'grow-row', 'grow-column', 'grow-fullscreen', 'rtl', 'timer-progress-bar', 'timer-progress-bar-container', 'scrollbar-measure', 'icon-success', 'icon-warning', 'icon-info', 'icon-question', 'icon-error']);
  const iconTypes = prefix(['success', 'warning', 'info', 'question', 'error']);

  const getContainer = () => document.body.querySelector(".".concat(swalClasses.container));
  const elementBySelector = selectorString => {
    const container = getContainer();
    return container ? container.querySelector(selectorString) : null;
  };

  const elementByClass = className => {
    return elementBySelector(".".concat(className));
  };

  const getPopup = () => elementByClass(swalClasses.popup);
  const getIcon = () => elementByClass(swalClasses.icon);
  const getTitle = () => elementByClass(swalClasses.title);
  const getHtmlContainer = () => elementByClass(swalClasses['html-container']);
  const getImage = () => elementByClass(swalClasses.image);
  const getProgressSteps = () => elementByClass(swalClasses['progress-steps']);
  const getValidationMessage = () => elementByClass(swalClasses['validation-message']);
  const getConfirmButton = () => elementBySelector(".".concat(swalClasses.actions, " .").concat(swalClasses.confirm));
  const getDenyButton = () => elementBySelector(".".concat(swalClasses.actions, " .").concat(swalClasses.deny));
  const getInputLabel = () => elementByClass(swalClasses['input-label']);
  const getLoader = () => elementBySelector(".".concat(swalClasses.loader));
  const getCancelButton = () => elementBySelector(".".concat(swalClasses.actions, " .").concat(swalClasses.cancel));
  const getActions = () => elementByClass(swalClasses.actions);
  const getFooter = () => elementByClass(swalClasses.footer);
  const getTimerProgressBar = () => elementByClass(swalClasses['timer-progress-bar']);
  const getCloseButton = () => elementByClass(swalClasses.close); // https://github.com/jkup/focusable/blob/master/index.js

  const focusable = "\n  a[href],\n  area[href],\n  input:not([disabled]),\n  select:not([disabled]),\n  textarea:not([disabled]),\n  button:not([disabled]),\n  iframe,\n  object,\n  embed,\n  [tabindex=\"0\"],\n  [contenteditable],\n  audio[controls],\n  video[controls],\n  summary\n";
  const getFocusableElements = () => {
    const focusableElementsWithTabindex = toArray(getPopup().querySelectorAll('[tabindex]:not([tabindex="-1"]):not([tabindex="0"])')) // sort according to tabindex
    .sort((a, b) => {
      a = parseInt(a.getAttribute('tabindex'));
      b = parseInt(b.getAttribute('tabindex'));

      if (a > b) {
        return 1;
      } else if (a < b) {
        return -1;
      }

      return 0;
    });
    const otherFocusableElements = toArray(getPopup().querySelectorAll(focusable)).filter(el => el.getAttribute('tabindex') !== '-1');
    return uniqueArray(focusableElementsWithTabindex.concat(otherFocusableElements)).filter(el => isVisible(el));
  };
  const isModal = () => {
    return !hasClass(document.body, swalClasses['toast-shown']) && !hasClass(document.body, swalClasses['no-backdrop']);
  };
  const isToast = () => {
    return getPopup() && hasClass(getPopup(), swalClasses.toast);
  };
  const isLoading = () => {
    return getPopup().hasAttribute('data-loading');
  };

  const states = {
    previousBodyPadding: null
  };
  const setInnerHtml = (elem, html) => {
    // #1926
    elem.textContent = '';

    if (html) {
      const parser = new DOMParser();
      const parsed = parser.parseFromString(html, "text/html");
      toArray(parsed.querySelector('head').childNodes).forEach(child => {
        elem.appendChild(child);
      });
      toArray(parsed.querySelector('body').childNodes).forEach(child => {
        elem.appendChild(child);
      });
    }
  };
  const hasClass = (elem, className) => {
    if (!className) {
      return false;
    }

    const classList = className.split(/\s+/);

    for (let i = 0; i < classList.length; i++) {
      if (!elem.classList.contains(classList[i])) {
        return false;
      }
    }

    return true;
  };

  const removeCustomClasses = (elem, params) => {
    toArray(elem.classList).forEach(className => {
      if (!Object.values(swalClasses).includes(className) && !Object.values(iconTypes).includes(className) && !Object.values(params.showClass).includes(className)) {
        elem.classList.remove(className);
      }
    });
  };

  const applyCustomClass = (elem, params, className) => {
    removeCustomClasses(elem, params);

    if (params.customClass && params.customClass[className]) {
      if (typeof params.customClass[className] !== 'string' && !params.customClass[className].forEach) {
        return warn("Invalid type of customClass.".concat(className, "! Expected string or iterable object, got \"").concat(typeof params.customClass[className], "\""));
      }

      addClass(elem, params.customClass[className]);
    }
  };
  const getInput = (popup, inputType) => {
    if (!inputType) {
      return null;
    }

    switch (inputType) {
      case 'select':
      case 'textarea':
      case 'file':
        return getChildByClass(popup, swalClasses[inputType]);

      case 'checkbox':
        return popup.querySelector(".".concat(swalClasses.checkbox, " input"));

      case 'radio':
        return popup.querySelector(".".concat(swalClasses.radio, " input:checked")) || popup.querySelector(".".concat(swalClasses.radio, " input:first-child"));

      case 'range':
        return popup.querySelector(".".concat(swalClasses.range, " input"));

      default:
        return getChildByClass(popup, swalClasses.input);
    }
  };
  const focusInput = input => {
    input.focus(); // place cursor at end of text in text input

    if (input.type !== 'file') {
      // http://stackoverflow.com/a/2345915
      const val = input.value;
      input.value = '';
      input.value = val;
    }
  };
  const toggleClass = (target, classList, condition) => {
    if (!target || !classList) {
      return;
    }

    if (typeof classList === 'string') {
      classList = classList.split(/\s+/).filter(Boolean);
    }

    classList.forEach(className => {
      if (target.forEach) {
        target.forEach(elem => {
          condition ? elem.classList.add(className) : elem.classList.remove(className);
        });
      } else {
        condition ? target.classList.add(className) : target.classList.remove(className);
      }
    });
  };
  const addClass = (target, classList) => {
    toggleClass(target, classList, true);
  };
  const removeClass = (target, classList) => {
    toggleClass(target, classList, false);
  };
  const getChildByClass = (elem, className) => {
    for (let i = 0; i < elem.childNodes.length; i++) {
      if (hasClass(elem.childNodes[i], className)) {
        return elem.childNodes[i];
      }
    }
  };
  const applyNumericalStyle = (elem, property, value) => {
    if (value === "".concat(parseInt(value))) {
      value = parseInt(value);
    }

    if (value || parseInt(value) === 0) {
      elem.style[property] = typeof value === 'number' ? "".concat(value, "px") : value;
    } else {
      elem.style.removeProperty(property);
    }
  };
  const show = function (elem) {
    let display = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : 'flex';
    elem.style.display = display;
  };
  const hide = elem => {
    elem.style.display = 'none';
  };
  const setStyle = (parent, selector, property, value) => {
    const el = parent.querySelector(selector);

    if (el) {
      el.style[property] = value;
    }
  };
  const toggle = (elem, condition, display) => {
    condition ? show(elem, display) : hide(elem);
  }; // borrowed from jquery $(elem).is(':visible') implementation

  const isVisible = elem => !!(elem && (elem.offsetWidth || elem.offsetHeight || elem.getClientRects().length));
  const allButtonsAreHidden = () => !isVisible(getConfirmButton()) && !isVisible(getDenyButton()) && !isVisible(getCancelButton());
  const isScrollable = elem => !!(elem.scrollHeight > elem.clientHeight); // borrowed from https://stackoverflow.com/a/46352119

  const hasCssAnimation = elem => {
    const style = window.getComputedStyle(elem);
    const animDuration = parseFloat(style.getPropertyValue('animation-duration') || '0');
    const transDuration = parseFloat(style.getPropertyValue('transition-duration') || '0');
    return animDuration > 0 || transDuration > 0;
  };
  const animateTimerProgressBar = function (timer) {
    let reset = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : false;
    const timerProgressBar = getTimerProgressBar();

    if (isVisible(timerProgressBar)) {
      if (reset) {
        timerProgressBar.style.transition = 'none';
        timerProgressBar.style.width = '100%';
      }

      setTimeout(() => {
        timerProgressBar.style.transition = "width ".concat(timer / 1000, "s linear");
        timerProgressBar.style.width = '0%';
      }, 10);
    }
  };
  const stopTimerProgressBar = () => {
    const timerProgressBar = getTimerProgressBar();
    const timerProgressBarWidth = parseInt(window.getComputedStyle(timerProgressBar).width);
    timerProgressBar.style.removeProperty('transition');
    timerProgressBar.style.width = '100%';
    const timerProgressBarFullWidth = parseInt(window.getComputedStyle(timerProgressBar).width);
    const timerProgressBarPercent = parseInt(timerProgressBarWidth / timerProgressBarFullWidth * 100);
    timerProgressBar.style.removeProperty('transition');
    timerProgressBar.style.width = "".concat(timerProgressBarPercent, "%");
  };

  // Detect Node env
  const isNodeEnv = () => typeof window === 'undefined' || typeof document === 'undefined';

  const sweetHTML = "\n <div aria-labelledby=\"".concat(swalClasses.title, "\" aria-describedby=\"").concat(swalClasses['html-container'], "\" class=\"").concat(swalClasses.popup, "\" tabindex=\"-1\">\n   <button type=\"button\" class=\"").concat(swalClasses.close, "\"></button>\n   <ul class=\"").concat(swalClasses['progress-steps'], "\"></ul>\n   <div class=\"").concat(swalClasses.icon, "\"></div>\n   <img class=\"").concat(swalClasses.image, "\" />\n   <h2 class=\"").concat(swalClasses.title, "\" id=\"").concat(swalClasses.title, "\"></h2>\n   <div class=\"").concat(swalClasses['html-container'], "\" id=\"").concat(swalClasses['html-container'], "\"></div>\n   <input class=\"").concat(swalClasses.input, "\" />\n   <input type=\"file\" class=\"").concat(swalClasses.file, "\" />\n   <div class=\"").concat(swalClasses.range, "\">\n     <input type=\"range\" />\n     <output></output>\n   </div>\n   <select class=\"").concat(swalClasses.select, "\"></select>\n   <div class=\"").concat(swalClasses.radio, "\"></div>\n   <label for=\"").concat(swalClasses.checkbox, "\" class=\"").concat(swalClasses.checkbox, "\">\n     <input type=\"checkbox\" />\n     <span class=\"").concat(swalClasses.label, "\"></span>\n   </label>\n   <textarea class=\"").concat(swalClasses.textarea, "\"></textarea>\n   <div class=\"").concat(swalClasses['validation-message'], "\" id=\"").concat(swalClasses['validation-message'], "\"></div>\n   <div class=\"").concat(swalClasses.actions, "\">\n     <div class=\"").concat(swalClasses.loader, "\"></div>\n     <button type=\"button\" class=\"").concat(swalClasses.confirm, "\"></button>\n     <button type=\"button\" class=\"").concat(swalClasses.deny, "\"></button>\n     <button type=\"button\" class=\"").concat(swalClasses.cancel, "\"></button>\n   </div>\n   <div class=\"").concat(swalClasses.footer, "\"></div>\n   <div class=\"").concat(swalClasses['timer-progress-bar-container'], "\">\n     <div class=\"").concat(swalClasses['timer-progress-bar'], "\"></div>\n   </div>\n </div>\n").replace(/(^|\n)\s*/g, '');

  const resetOldContainer = () => {
    const oldContainer = getContainer();

    if (!oldContainer) {
      return false;
    }

    oldContainer.remove();
    removeClass([document.documentElement, document.body], [swalClasses['no-backdrop'], swalClasses['toast-shown'], swalClasses['has-column']]);
    return true;
  };

  const resetValidationMessage = () => {
    if (Swal.isVisible()) {
      Swal.resetValidationMessage();
    }
  };

  const addInputChangeListeners = () => {
    const popup = getPopup();
    const input = getChildByClass(popup, swalClasses.input);
    const file = getChildByClass(popup, swalClasses.file);
    const range = popup.querySelector(".".concat(swalClasses.range, " input"));
    const rangeOutput = popup.querySelector(".".concat(swalClasses.range, " output"));
    const select = getChildByClass(popup, swalClasses.select);
    const checkbox = popup.querySelector(".".concat(swalClasses.checkbox, " input"));
    const textarea = getChildByClass(popup, swalClasses.textarea);
    input.oninput = resetValidationMessage;
    file.onchange = resetValidationMessage;
    select.onchange = resetValidationMessage;
    checkbox.onchange = resetValidationMessage;
    textarea.oninput = resetValidationMessage;

    range.oninput = () => {
      resetValidationMessage();
      rangeOutput.value = range.value;
    };

    range.onchange = () => {
      resetValidationMessage();
      range.nextSibling.value = range.value;
    };
  };

  const getTarget = target => typeof target === 'string' ? document.querySelector(target) : target;

  const setupAccessibility = params => {
    const popup = getPopup();
    popup.setAttribute('role', params.toast ? 'alert' : 'dialog');
    popup.setAttribute('aria-live', params.toast ? 'polite' : 'assertive');

    if (!params.toast) {
      popup.setAttribute('aria-modal', 'true');
    }
  };

  const setupRTL = targetElement => {
    if (window.getComputedStyle(targetElement).direction === 'rtl') {
      addClass(getContainer(), swalClasses.rtl);
    }
  };
  /*
   * Add modal + backdrop to DOM
   */


  const init = params => {
    // Clean up the old popup container if it exists
    const oldContainerExisted = resetOldContainer();
    /* istanbul ignore if */

    if (isNodeEnv()) {
      error('SweetAlert2 requires document to initialize');
      return;
    }

    const container = document.createElement('div');
    container.className = swalClasses.container;

    if (oldContainerExisted) {
      addClass(container, swalClasses['no-transition']);
    }

    setInnerHtml(container, sweetHTML);
    const targetElement = getTarget(params.target);
    targetElement.appendChild(container);
    setupAccessibility(params);
    setupRTL(targetElement);
    addInputChangeListeners();
  };

  const parseHtmlToContainer = (param, target) => {
    // DOM element
    if (param instanceof HTMLElement) {
      target.appendChild(param); // Object
    } else if (typeof param === 'object') {
      handleObject(param, target); // Plain string
    } else if (param) {
      setInnerHtml(target, param);
    }
  };

  const handleObject = (param, target) => {
    // JQuery element(s)
    if (param.jquery) {
      handleJqueryElem(target, param); // For other objects use their string representation
    } else {
      setInnerHtml(target, param.toString());
    }
  };

  const handleJqueryElem = (target, elem) => {
    target.textContent = '';

    if (0 in elem) {
      for (let i = 0; (i in elem); i++) {
        target.appendChild(elem[i].cloneNode(true));
      }
    } else {
      target.appendChild(elem.cloneNode(true));
    }
  };

  const animationEndEvent = (() => {
    // Prevent run in Node env

    /* istanbul ignore if */
    if (isNodeEnv()) {
      return false;
    }

    const testEl = document.createElement('div');
    const transEndEventNames = {
      WebkitAnimation: 'webkitAnimationEnd',
      OAnimation: 'oAnimationEnd oanimationend',
      animation: 'animationend'
    };

    for (const i in transEndEventNames) {
      if (Object.prototype.hasOwnProperty.call(transEndEventNames, i) && typeof testEl.style[i] !== 'undefined') {
        return transEndEventNames[i];
      }
    }

    return false;
  })();

  // https://github.com/twbs/bootstrap/blob/master/js/src/modal.js

  const measureScrollbar = () => {
    const scrollDiv = document.createElement('div');
    scrollDiv.className = swalClasses['scrollbar-measure'];
    document.body.appendChild(scrollDiv);
    const scrollbarWidth = scrollDiv.getBoundingClientRect().width - scrollDiv.clientWidth;
    document.body.removeChild(scrollDiv);
    return scrollbarWidth;
  };

  const renderActions = (instance, params) => {
    const actions = getActions();
    const loader = getLoader(); // Actions (buttons) wrapper

    if (!params.showConfirmButton && !params.showDenyButton && !params.showCancelButton) {
      hide(actions);
    } else {
      show(actions);
    } // Custom class


    applyCustomClass(actions, params, 'actions'); // Render all the buttons

    renderButtons(actions, loader, params); // Loader

    setInnerHtml(loader, params.loaderHtml);
    applyCustomClass(loader, params, 'loader');
  };

  function renderButtons(actions, loader, params) {
    const confirmButton = getConfirmButton();
    const denyButton = getDenyButton();
    const cancelButton = getCancelButton(); // Render buttons

    renderButton(confirmButton, 'confirm', params);
    renderButton(denyButton, 'deny', params);
    renderButton(cancelButton, 'cancel', params);
    handleButtonsStyling(confirmButton, denyButton, cancelButton, params);

    if (params.reverseButtons) {
      if (params.toast) {
        actions.insertBefore(cancelButton, confirmButton);
        actions.insertBefore(denyButton, confirmButton);
      } else {
        actions.insertBefore(cancelButton, loader);
        actions.insertBefore(denyButton, loader);
        actions.insertBefore(confirmButton, loader);
      }
    }
  }

  function handleButtonsStyling(confirmButton, denyButton, cancelButton, params) {
    if (!params.buttonsStyling) {
      return removeClass([confirmButton, denyButton, cancelButton], swalClasses.styled);
    }

    addClass([confirmButton, denyButton, cancelButton], swalClasses.styled); // Buttons background colors

    if (params.confirmButtonColor) {
      confirmButton.style.backgroundColor = params.confirmButtonColor;
      addClass(confirmButton, swalClasses['default-outline']);
    }

    if (params.denyButtonColor) {
      denyButton.style.backgroundColor = params.denyButtonColor;
      addClass(denyButton, swalClasses['default-outline']);
    }

    if (params.cancelButtonColor) {
      cancelButton.style.backgroundColor = params.cancelButtonColor;
      addClass(cancelButton, swalClasses['default-outline']);
    }
  }

  function renderButton(button, buttonType, params) {
    toggle(button, params["show".concat(capitalizeFirstLetter(buttonType), "Button")], 'inline-block');
    setInnerHtml(button, params["".concat(buttonType, "ButtonText")]); // Set caption text

    button.setAttribute('aria-label', params["".concat(buttonType, "ButtonAriaLabel")]); // ARIA label
    // Add buttons custom classes

    button.className = swalClasses[buttonType];
    applyCustomClass(button, params, "".concat(buttonType, "Button"));
    addClass(button, params["".concat(buttonType, "ButtonClass")]);
  }

  function handleBackdropParam(container, backdrop) {
    if (typeof backdrop === 'string') {
      container.style.background = backdrop;
    } else if (!backdrop) {
      addClass([document.documentElement, document.body], swalClasses['no-backdrop']);
    }
  }

  function handlePositionParam(container, position) {
    if (position in swalClasses) {
      addClass(container, swalClasses[position]);
    } else {
      warn('The "position" parameter is not valid, defaulting to "center"');
      addClass(container, swalClasses.center);
    }
  }

  function handleGrowParam(container, grow) {
    if (grow && typeof grow === 'string') {
      const growClass = "grow-".concat(grow);

      if (growClass in swalClasses) {
        addClass(container, swalClasses[growClass]);
      }
    }
  }

  const renderContainer = (instance, params) => {
    const container = getContainer();

    if (!container) {
      return;
    }

    handleBackdropParam(container, params.backdrop);
    handlePositionParam(container, params.position);
    handleGrowParam(container, params.grow); // Custom class

    applyCustomClass(container, params, 'container');
  };

  /**
   * This module contains `WeakMap`s for each effectively-"private  property" that a `Swal` has.
   * For example, to set the private property "foo" of `this` to "bar", you can `privateProps.foo.set(this, 'bar')`
   * This is the approach that Babel will probably take to implement private methods/fields
   *   https://github.com/tc39/proposal-private-methods
   *   https://github.com/babel/babel/pull/7555
   * Once we have the changes from that PR in Babel, and our core class fits reasonable in *one module*
   *   then we can use that language feature.
   */
  var privateProps = {
    awaitingPromise: new WeakMap(),
    promise: new WeakMap(),
    innerParams: new WeakMap(),
    domCache: new WeakMap()
  };

  const inputTypes = ['input', 'file', 'range', 'select', 'radio', 'checkbox', 'textarea'];
  const renderInput = (instance, params) => {
    const popup = getPopup();
    const innerParams = privateProps.innerParams.get(instance);
    const rerender = !innerParams || params.input !== innerParams.input;
    inputTypes.forEach(inputType => {
      const inputClass = swalClasses[inputType];
      const inputContainer = getChildByClass(popup, inputClass); // set attributes

      setAttributes(inputType, params.inputAttributes); // set class

      inputContainer.className = inputClass;

      if (rerender) {
        hide(inputContainer);
      }
    });

    if (params.input) {
      if (rerender) {
        showInput(params);
      } // set custom class


      setCustomClass(params);
    }
  };

  const showInput = params => {
    if (!renderInputType[params.input]) {
      return error("Unexpected type of input! Expected \"text\", \"email\", \"password\", \"number\", \"tel\", \"select\", \"radio\", \"checkbox\", \"textarea\", \"file\" or \"url\", got \"".concat(params.input, "\""));
    }

    const inputContainer = getInputContainer(params.input);
    const input = renderInputType[params.input](inputContainer, params);
    show(input); // input autofocus

    setTimeout(() => {
      focusInput(input);
    });
  };

  const removeAttributes = input => {
    for (let i = 0; i < input.attributes.length; i++) {
      const attrName = input.attributes[i].name;

      if (!['type', 'value', 'style'].includes(attrName)) {
        input.removeAttribute(attrName);
      }
    }
  };

  const setAttributes = (inputType, inputAttributes) => {
    const input = getInput(getPopup(), inputType);

    if (!input) {
      return;
    }

    removeAttributes(input);

    for (const attr in inputAttributes) {
      input.setAttribute(attr, inputAttributes[attr]);
    }
  };

  const setCustomClass = params => {
    const inputContainer = getInputContainer(params.input);

    if (params.customClass) {
      addClass(inputContainer, params.customClass.input);
    }
  };

  const setInputPlaceholder = (input, params) => {
    if (!input.placeholder || params.inputPlaceholder) {
      input.placeholder = params.inputPlaceholder;
    }
  };

  const setInputLabel = (input, prependTo, params) => {
    if (params.inputLabel) {
      input.id = swalClasses.input;
      const label = document.createElement('label');
      const labelClass = swalClasses['input-label'];
      label.setAttribute('for', input.id);
      label.className = labelClass;
      addClass(label, params.customClass.inputLabel);
      label.innerText = params.inputLabel;
      prependTo.insertAdjacentElement('beforebegin', label);
    }
  };

  const getInputContainer = inputType => {
    const inputClass = swalClasses[inputType] ? swalClasses[inputType] : swalClasses.input;
    return getChildByClass(getPopup(), inputClass);
  };

  const renderInputType = {};

  renderInputType.text = renderInputType.email = renderInputType.password = renderInputType.number = renderInputType.tel = renderInputType.url = (input, params) => {
    if (typeof params.inputValue === 'string' || typeof params.inputValue === 'number') {
      input.value = params.inputValue;
    } else if (!isPromise(params.inputValue)) {
      warn("Unexpected type of inputValue! Expected \"string\", \"number\" or \"Promise\", got \"".concat(typeof params.inputValue, "\""));
    }

    setInputLabel(input, input, params);
    setInputPlaceholder(input, params);
    input.type = params.input;
    return input;
  };

  renderInputType.file = (input, params) => {
    setInputLabel(input, input, params);
    setInputPlaceholder(input, params);
    return input;
  };

  renderInputType.range = (range, params) => {
    const rangeInput = range.querySelector('input');
    const rangeOutput = range.querySelector('output');
    rangeInput.value = params.inputValue;
    rangeInput.type = params.input;
    rangeOutput.value = params.inputValue;
    setInputLabel(rangeInput, range, params);
    return range;
  };

  renderInputType.select = (select, params) => {
    select.textContent = '';

    if (params.inputPlaceholder) {
      const placeholder = document.createElement('option');
      setInnerHtml(placeholder, params.inputPlaceholder);
      placeholder.value = '';
      placeholder.disabled = true;
      placeholder.selected = true;
      select.appendChild(placeholder);
    }

    setInputLabel(select, select, params);
    return select;
  };

  renderInputType.radio = radio => {
    radio.textContent = '';
    return radio;
  };

  renderInputType.checkbox = (checkboxContainer, params) => {
    const checkbox = getInput(getPopup(), 'checkbox');
    checkbox.value = 1;
    checkbox.id = swalClasses.checkbox;
    checkbox.checked = Boolean(params.inputValue);
    const label = checkboxContainer.querySelector('span');
    setInnerHtml(label, params.inputPlaceholder);
    return checkboxContainer;
  };

  renderInputType.textarea = (textarea, params) => {
    textarea.value = params.inputValue;
    setInputPlaceholder(textarea, params);
    setInputLabel(textarea, textarea, params);

    const getMargin = el => parseInt(window.getComputedStyle(el).marginLeft) + parseInt(window.getComputedStyle(el).marginRight);

    setTimeout(() => {
      // #2291
      if ('MutationObserver' in window) {
        // #1699
        const initialPopupWidth = parseInt(window.getComputedStyle(getPopup()).width);

        const textareaResizeHandler = () => {
          const textareaWidth = textarea.offsetWidth + getMargin(textarea);

          if (textareaWidth > initialPopupWidth) {
            getPopup().style.width = "".concat(textareaWidth, "px");
          } else {
            getPopup().style.width = null;
          }
        };

        new MutationObserver(textareaResizeHandler).observe(textarea, {
          attributes: true,
          attributeFilter: ['style']
        });
      }
    });
    return textarea;
  };

  const renderContent = (instance, params) => {
    const htmlContainer = getHtmlContainer();
    applyCustomClass(htmlContainer, params, 'htmlContainer'); // Content as HTML

    if (params.html) {
      parseHtmlToContainer(params.html, htmlContainer);
      show(htmlContainer, 'block'); // Content as plain text
    } else if (params.text) {
      htmlContainer.textContent = params.text;
      show(htmlContainer, 'block'); // No content
    } else {
      hide(htmlContainer);
    }

    renderInput(instance, params);
  };

  const renderFooter = (instance, params) => {
    const footer = getFooter();
    toggle(footer, params.footer);

    if (params.footer) {
      parseHtmlToContainer(params.footer, footer);
    } // Custom class


    applyCustomClass(footer, params, 'footer');
  };

  const renderCloseButton = (instance, params) => {
    const closeButton = getCloseButton();
    setInnerHtml(closeButton, params.closeButtonHtml); // Custom class

    applyCustomClass(closeButton, params, 'closeButton');
    toggle(closeButton, params.showCloseButton);
    closeButton.setAttribute('aria-label', params.closeButtonAriaLabel);
  };

  const renderIcon = (instance, params) => {
    const innerParams = privateProps.innerParams.get(instance);
    const icon = getIcon(); // if the given icon already rendered, apply the styling without re-rendering the icon

    if (innerParams && params.icon === innerParams.icon) {
      // Custom or default content
      setContent(icon, params);
      applyStyles(icon, params);
      return;
    }

    if (!params.icon && !params.iconHtml) {
      return hide(icon);
    }

    if (params.icon && Object.keys(iconTypes).indexOf(params.icon) === -1) {
      error("Unknown icon! Expected \"success\", \"error\", \"warning\", \"info\" or \"question\", got \"".concat(params.icon, "\""));
      return hide(icon);
    }

    show(icon); // Custom or default content

    setContent(icon, params);
    applyStyles(icon, params); // Animate icon

    addClass(icon, params.showClass.icon);
  };

  const applyStyles = (icon, params) => {
    for (const iconType in iconTypes) {
      if (params.icon !== iconType) {
        removeClass(icon, iconTypes[iconType]);
      }
    }

    addClass(icon, iconTypes[params.icon]); // Icon color

    setColor(icon, params); // Success icon background color

    adjustSuccessIconBackgoundColor(); // Custom class

    applyCustomClass(icon, params, 'icon');
  }; // Adjust success icon background color to match the popup background color


  const adjustSuccessIconBackgoundColor = () => {
    const popup = getPopup();
    const popupBackgroundColor = window.getComputedStyle(popup).getPropertyValue('background-color');
    const successIconParts = popup.querySelectorAll('[class^=swal2-success-circular-line], .swal2-success-fix');

    for (let i = 0; i < successIconParts.length; i++) {
      successIconParts[i].style.backgroundColor = popupBackgroundColor;
    }
  };

  const setContent = (icon, params) => {
    icon.textContent = '';

    if (params.iconHtml) {
      setInnerHtml(icon, iconContent(params.iconHtml));
    } else if (params.icon === 'success') {
      setInnerHtml(icon, "\n      <div class=\"swal2-success-circular-line-left\"></div>\n      <span class=\"swal2-success-line-tip\"></span> <span class=\"swal2-success-line-long\"></span>\n      <div class=\"swal2-success-ring\"></div> <div class=\"swal2-success-fix\"></div>\n      <div class=\"swal2-success-circular-line-right\"></div>\n    ");
    } else if (params.icon === 'error') {
      setInnerHtml(icon, "\n      <span class=\"swal2-x-mark\">\n        <span class=\"swal2-x-mark-line-left\"></span>\n        <span class=\"swal2-x-mark-line-right\"></span>\n      </span>\n    ");
    } else {
      const defaultIconHtml = {
        question: '?',
        warning: '!',
        info: 'i'
      };
      setInnerHtml(icon, iconContent(defaultIconHtml[params.icon]));
    }
  };

  const setColor = (icon, params) => {
    if (!params.iconColor) {
      return;
    }

    icon.style.color = params.iconColor;
    icon.style.borderColor = params.iconColor;

    for (const sel of ['.swal2-success-line-tip', '.swal2-success-line-long', '.swal2-x-mark-line-left', '.swal2-x-mark-line-right']) {
      setStyle(icon, sel, 'backgroundColor', params.iconColor);
    }

    setStyle(icon, '.swal2-success-ring', 'borderColor', params.iconColor);
  };

  const iconContent = content => "<div class=\"".concat(swalClasses['icon-content'], "\">").concat(content, "</div>");

  const renderImage = (instance, params) => {
    const image = getImage();

    if (!params.imageUrl) {
      return hide(image);
    }

    show(image, ''); // Src, alt

    image.setAttribute('src', params.imageUrl);
    image.setAttribute('alt', params.imageAlt); // Width, height

    applyNumericalStyle(image, 'width', params.imageWidth);
    applyNumericalStyle(image, 'height', params.imageHeight); // Class

    image.className = swalClasses.image;
    applyCustomClass(image, params, 'image');
  };

  const createStepElement = step => {
    const stepEl = document.createElement('li');
    addClass(stepEl, swalClasses['progress-step']);
    setInnerHtml(stepEl, step);
    return stepEl;
  };

  const createLineElement = params => {
    const lineEl = document.createElement('li');
    addClass(lineEl, swalClasses['progress-step-line']);

    if (params.progressStepsDistance) {
      lineEl.style.width = params.progressStepsDistance;
    }

    return lineEl;
  };

  const renderProgressSteps = (instance, params) => {
    const progressStepsContainer = getProgressSteps();

    if (!params.progressSteps || params.progressSteps.length === 0) {
      return hide(progressStepsContainer);
    }

    show(progressStepsContainer);
    progressStepsContainer.textContent = '';

    if (params.currentProgressStep >= params.progressSteps.length) {
      warn('Invalid currentProgressStep parameter, it should be less than progressSteps.length ' + '(currentProgressStep like JS arrays starts from 0)');
    }

    params.progressSteps.forEach((step, index) => {
      const stepEl = createStepElement(step);
      progressStepsContainer.appendChild(stepEl);

      if (index === params.currentProgressStep) {
        addClass(stepEl, swalClasses['active-progress-step']);
      }

      if (index !== params.progressSteps.length - 1) {
        const lineEl = createLineElement(params);
        progressStepsContainer.appendChild(lineEl);
      }
    });
  };

  const renderTitle = (instance, params) => {
    const title = getTitle();
    toggle(title, params.title || params.titleText, 'block');

    if (params.title) {
      parseHtmlToContainer(params.title, title);
    }

    if (params.titleText) {
      title.innerText = params.titleText;
    } // Custom class


    applyCustomClass(title, params, 'title');
  };

  const renderPopup = (instance, params) => {
    const container = getContainer();
    const popup = getPopup(); // Width

    if (params.toast) {
      // #2170
      applyNumericalStyle(container, 'width', params.width);
      popup.style.width = '100%';
      popup.insertBefore(getLoader(), getIcon());
    } else {
      applyNumericalStyle(popup, 'width', params.width);
    } // Padding


    applyNumericalStyle(popup, 'padding', params.padding); // Color

    if (params.color) {
      popup.style.color = params.color;
    } // Background


    if (params.background) {
      popup.style.background = params.background;
    }

    hide(getValidationMessage()); // Classes

    addClasses(popup, params);
  };

  const addClasses = (popup, params) => {
    // Default Class + showClass when updating Swal.update({})
    popup.className = "".concat(swalClasses.popup, " ").concat(isVisible(popup) ? params.showClass.popup : '');

    if (params.toast) {
      addClass([document.documentElement, document.body], swalClasses['toast-shown']);
      addClass(popup, swalClasses.toast);
    } else {
      addClass(popup, swalClasses.modal);
    } // Custom class


    applyCustomClass(popup, params, 'popup');

    if (typeof params.customClass === 'string') {
      addClass(popup, params.customClass);
    } // Icon class (#1842)


    if (params.icon) {
      addClass(popup, swalClasses["icon-".concat(params.icon)]);
    }
  };

  const render = (instance, params) => {
    renderPopup(instance, params);
    renderContainer(instance, params);
    renderProgressSteps(instance, params);
    renderIcon(instance, params);
    renderImage(instance, params);
    renderTitle(instance, params);
    renderCloseButton(instance, params);
    renderContent(instance, params);
    renderActions(instance, params);
    renderFooter(instance, params);

    if (typeof params.didRender === 'function') {
      params.didRender(getPopup());
    }
  };

  /*
   * Global function to determine if SweetAlert2 popup is shown
   */

  const isVisible$1 = () => {
    return isVisible(getPopup());
  };
  /*
   * Global function to click 'Confirm' button
   */

  const clickConfirm = () => getConfirmButton() && getConfirmButton().click();
  /*
   * Global function to click 'Deny' button
   */

  const clickDeny = () => getDenyButton() && getDenyButton().click();
  /*
   * Global function to click 'Cancel' button
   */

  const clickCancel = () => getCancelButton() && getCancelButton().click();

  function fire() {
    const Swal = this;

    for (var _len = arguments.length, args = new Array(_len), _key = 0; _key < _len; _key++) {
      args[_key] = arguments[_key];
    }

    return new Swal(...args);
  }

  /**
   * Returns an extended version of `Swal` containing `params` as defaults.
   * Useful for reusing Swal configuration.
   *
   * For example:
   *
   * Before:
   * const textPromptOptions = { input: 'text', showCancelButton: true }
   * const {value: firstName} = await Swal.fire({ ...textPromptOptions, title: 'What is your first name?' })
   * const {value: lastName} = await Swal.fire({ ...textPromptOptions, title: 'What is your last name?' })
   *
   * After:
   * const TextPrompt = Swal.mixin({ input: 'text', showCancelButton: true })
   * const {value: firstName} = await TextPrompt('What is your first name?')
   * const {value: lastName} = await TextPrompt('What is your last name?')
   *
   * @param mixinParams
   */
  function mixin(mixinParams) {
    class MixinSwal extends this {
      _main(params, priorityMixinParams) {
        return super._main(params, Object.assign({}, mixinParams, priorityMixinParams));
      }

    }

    return MixinSwal;
  }

  /**
   * Shows loader (spinner), this is useful with AJAX requests.
   * By default the loader be shown instead of the "Confirm" button.
   */

  const showLoading = buttonToReplace => {
    let popup = getPopup();

    if (!popup) {
      Swal.fire();
    }

    popup = getPopup();
    const loader = getLoader();

    if (isToast()) {
      hide(getIcon());
    } else {
      replaceButton(popup, buttonToReplace);
    }

    show(loader);
    popup.setAttribute('data-loading', true);
    popup.setAttribute('aria-busy', true);
    popup.focus();
  };

  const replaceButton = (popup, buttonToReplace) => {
    const actions = getActions();
    const loader = getLoader();

    if (!buttonToReplace && isVisible(getConfirmButton())) {
      buttonToReplace = getConfirmButton();
    }

    show(actions);

    if (buttonToReplace) {
      hide(buttonToReplace);
      loader.setAttribute('data-button-to-replace', buttonToReplace.className);
    }

    loader.parentNode.insertBefore(loader, buttonToReplace);
    addClass([popup, actions], swalClasses.loading);
  };

  const RESTORE_FOCUS_TIMEOUT = 100;

  const globalState = {};

  const focusPreviousActiveElement = () => {
    if (globalState.previousActiveElement && globalState.previousActiveElement.focus) {
      globalState.previousActiveElement.focus();
      globalState.previousActiveElement = null;
    } else if (document.body) {
      document.body.focus();
    }
  }; // Restore previous active (focused) element


  const restoreActiveElement = returnFocus => {
    return new Promise(resolve => {
      if (!returnFocus) {
        return resolve();
      }

      const x = window.scrollX;
      const y = window.scrollY;
      globalState.restoreFocusTimeout = setTimeout(() => {
        focusPreviousActiveElement();
        resolve();
      }, RESTORE_FOCUS_TIMEOUT); // issues/900

      window.scrollTo(x, y);
    });
  };

  /**
   * If `timer` parameter is set, returns number of milliseconds of timer remained.
   * Otherwise, returns undefined.
   */

  const getTimerLeft = () => {
    return globalState.timeout && globalState.timeout.getTimerLeft();
  };
  /**
   * Stop timer. Returns number of milliseconds of timer remained.
   * If `timer` parameter isn't set, returns undefined.
   */

  const stopTimer = () => {
    if (globalState.timeout) {
      stopTimerProgressBar();
      return globalState.timeout.stop();
    }
  };
  /**
   * Resume timer. Returns number of milliseconds of timer remained.
   * If `timer` parameter isn't set, returns undefined.
   */

  const resumeTimer = () => {
    if (globalState.timeout) {
      const remaining = globalState.timeout.start();
      animateTimerProgressBar(remaining);
      return remaining;
    }
  };
  /**
   * Resume timer. Returns number of milliseconds of timer remained.
   * If `timer` parameter isn't set, returns undefined.
   */

  const toggleTimer = () => {
    const timer = globalState.timeout;
    return timer && (timer.running ? stopTimer() : resumeTimer());
  };
  /**
   * Increase timer. Returns number of milliseconds of an updated timer.
   * If `timer` parameter isn't set, returns undefined.
   */

  const increaseTimer = n => {
    if (globalState.timeout) {
      const remaining = globalState.timeout.increase(n);
      animateTimerProgressBar(remaining, true);
      return remaining;
    }
  };
  /**
   * Check if timer is running. Returns true if timer is running
   * or false if timer is paused or stopped.
   * If `timer` parameter isn't set, returns undefined
   */

  const isTimerRunning = () => {
    return globalState.timeout && globalState.timeout.isRunning();
  };

  let bodyClickListenerAdded = false;
  const clickHandlers = {};
  function bindClickHandler() {
    let attr = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : 'data-swal-template';
    clickHandlers[attr] = this;

    if (!bodyClickListenerAdded) {
      document.body.addEventListener('click', bodyClickListener);
      bodyClickListenerAdded = true;
    }
  }

  const bodyClickListener = event => {
    for (let el = event.target; el && el !== document; el = el.parentNode) {
      for (const attr in clickHandlers) {
        const template = el.getAttribute(attr);

        if (template) {
          clickHandlers[attr].fire({
            template
          });
          return;
        }
      }
    }
  };

  const defaultParams = {
    title: '',
    titleText: '',
    text: '',
    html: '',
    footer: '',
    icon: undefined,
    iconColor: undefined,
    iconHtml: undefined,
    template: undefined,
    toast: false,
    showClass: {
      popup: 'swal2-show',
      backdrop: 'swal2-backdrop-show',
      icon: 'swal2-icon-show'
    },
    hideClass: {
      popup: 'swal2-hide',
      backdrop: 'swal2-backdrop-hide',
      icon: 'swal2-icon-hide'
    },
    customClass: {},
    target: 'body',
    color: undefined,
    backdrop: true,
    heightAuto: true,
    allowOutsideClick: true,
    allowEscapeKey: true,
    allowEnterKey: true,
    stopKeydownPropagation: true,
    keydownListenerCapture: false,
    showConfirmButton: true,
    showDenyButton: false,
    showCancelButton: false,
    preConfirm: undefined,
    preDeny: undefined,
    confirmButtonText: 'OK',
    confirmButtonAriaLabel: '',
    confirmButtonColor: undefined,
    denyButtonText: 'No',
    denyButtonAriaLabel: '',
    denyButtonColor: undefined,
    cancelButtonText: 'Cancel',
    cancelButtonAriaLabel: '',
    cancelButtonColor: undefined,
    buttonsStyling: true,
    reverseButtons: false,
    focusConfirm: true,
    focusDeny: false,
    focusCancel: false,
    returnFocus: true,
    showCloseButton: false,
    closeButtonHtml: '&times;',
    closeButtonAriaLabel: 'Close this dialog',
    loaderHtml: '',
    showLoaderOnConfirm: false,
    showLoaderOnDeny: false,
    imageUrl: undefined,
    imageWidth: undefined,
    imageHeight: undefined,
    imageAlt: '',
    timer: undefined,
    timerProgressBar: false,
    width: undefined,
    padding: undefined,
    background: undefined,
    input: undefined,
    inputPlaceholder: '',
    inputLabel: '',
    inputValue: '',
    inputOptions: {},
    inputAutoTrim: true,
    inputAttributes: {},
    inputValidator: undefined,
    returnInputValueOnDeny: false,
    validationMessage: undefined,
    grow: false,
    position: 'center',
    progressSteps: [],
    currentProgressStep: undefined,
    progressStepsDistance: undefined,
    willOpen: undefined,
    didOpen: undefined,
    didRender: undefined,
    willClose: undefined,
    didClose: undefined,
    didDestroy: undefined,
    scrollbarPadding: true
  };
  const updatableParams = ['allowEscapeKey', 'allowOutsideClick', 'background', 'buttonsStyling', 'cancelButtonAriaLabel', 'cancelButtonColor', 'cancelButtonText', 'closeButtonAriaLabel', 'closeButtonHtml', 'color', 'confirmButtonAriaLabel', 'confirmButtonColor', 'confirmButtonText', 'currentProgressStep', 'customClass', 'denyButtonAriaLabel', 'denyButtonColor', 'denyButtonText', 'didClose', 'didDestroy', 'footer', 'hideClass', 'html', 'icon', 'iconColor', 'iconHtml', 'imageAlt', 'imageHeight', 'imageUrl', 'imageWidth', 'preConfirm', 'preDeny', 'progressSteps', 'returnFocus', 'reverseButtons', 'showCancelButton', 'showCloseButton', 'showConfirmButton', 'showDenyButton', 'text', 'title', 'titleText', 'willClose'];
  const deprecatedParams = {};
  const toastIncompatibleParams = ['allowOutsideClick', 'allowEnterKey', 'backdrop', 'focusConfirm', 'focusDeny', 'focusCancel', 'returnFocus', 'heightAuto', 'keydownListenerCapture'];
  /**
   * Is valid parameter
   * @param {String} paramName
   */

  const isValidParameter = paramName => {
    return Object.prototype.hasOwnProperty.call(defaultParams, paramName);
  };
  /**
   * Is valid parameter for Swal.update() method
   * @param {String} paramName
   */

  const isUpdatableParameter = paramName => {
    return updatableParams.indexOf(paramName) !== -1;
  };
  /**
   * Is deprecated parameter
   * @param {String} paramName
   */

  const isDeprecatedParameter = paramName => {
    return deprecatedParams[paramName];
  };

  const checkIfParamIsValid = param => {
    if (!isValidParameter(param)) {
      warn("Unknown parameter \"".concat(param, "\""));
    }
  };

  const checkIfToastParamIsValid = param => {
    if (toastIncompatibleParams.includes(param)) {
      warn("The parameter \"".concat(param, "\" is incompatible with toasts"));
    }
  };

  const checkIfParamIsDeprecated = param => {
    if (isDeprecatedParameter(param)) {
      warnAboutDeprecation(param, isDeprecatedParameter(param));
    }
  };
  /**
   * Show relevant warnings for given params
   *
   * @param params
   */


  const showWarningsForParams = params => {
    if (!params.backdrop && params.allowOutsideClick) {
      warn('"allowOutsideClick" parameter requires `backdrop` parameter to be set to `true`');
    }

    for (const param in params) {
      checkIfParamIsValid(param);

      if (params.toast) {
        checkIfToastParamIsValid(param);
      }

      checkIfParamIsDeprecated(param);
    }
  };



  var staticMethods = /*#__PURE__*/Object.freeze({
    isValidParameter: isValidParameter,
    isUpdatableParameter: isUpdatableParameter,
    isDeprecatedParameter: isDeprecatedParameter,
    argsToParams: argsToParams,
    isVisible: isVisible$1,
    clickConfirm: clickConfirm,
    clickDeny: clickDeny,
    clickCancel: clickCancel,
    getContainer: getContainer,
    getPopup: getPopup,
    getTitle: getTitle,
    getHtmlContainer: getHtmlContainer,
    getImage: getImage,
    getIcon: getIcon,
    getInputLabel: getInputLabel,
    getCloseButton: getCloseButton,
    getActions: getActions,
    getConfirmButton: getConfirmButton,
    getDenyButton: getDenyButton,
    getCancelButton: getCancelButton,
    getLoader: getLoader,
    getFooter: getFooter,
    getTimerProgressBar: getTimerProgressBar,
    getFocusableElements: getFocusableElements,
    getValidationMessage: getValidationMessage,
    isLoading: isLoading,
    fire: fire,
    mixin: mixin,
    showLoading: showLoading,
    enableLoading: showLoading,
    getTimerLeft: getTimerLeft,
    stopTimer: stopTimer,
    resumeTimer: resumeTimer,
    toggleTimer: toggleTimer,
    increaseTimer: increaseTimer,
    isTimerRunning: isTimerRunning,
    bindClickHandler: bindClickHandler
  });

  /**
   * Hides loader and shows back the button which was hidden by .showLoading()
   */

  function hideLoading() {
    // do nothing if popup is closed
    const innerParams = privateProps.innerParams.get(this);

    if (!innerParams) {
      return;
    }

    const domCache = privateProps.domCache.get(this);
    hide(domCache.loader);

    if (isToast()) {
      if (innerParams.icon) {
        show(getIcon());
      }
    } else {
      showRelatedButton(domCache);
    }

    removeClass([domCache.popup, domCache.actions], swalClasses.loading);
    domCache.popup.removeAttribute('aria-busy');
    domCache.popup.removeAttribute('data-loading');
    domCache.confirmButton.disabled = false;
    domCache.denyButton.disabled = false;
    domCache.cancelButton.disabled = false;
  }

  const showRelatedButton = domCache => {
    const buttonToReplace = domCache.popup.getElementsByClassName(domCache.loader.getAttribute('data-button-to-replace'));

    if (buttonToReplace.length) {
      show(buttonToReplace[0], 'inline-block');
    } else if (allButtonsAreHidden()) {
      hide(domCache.actions);
    }
  };

  function getInput$1(instance) {
    const innerParams = privateProps.innerParams.get(instance || this);
    const domCache = privateProps.domCache.get(instance || this);

    if (!domCache) {
      return null;
    }

    return getInput(domCache.popup, innerParams.input);
  }

  const fixScrollbar = () => {
    // for queues, do not do this more than once
    if (states.previousBodyPadding !== null) {
      return;
    } // if the body has overflow


    if (document.body.scrollHeight > window.innerHeight) {
      // add padding so the content doesn't shift after removal of scrollbar
      states.previousBodyPadding = parseInt(window.getComputedStyle(document.body).getPropertyValue('padding-right'));
      document.body.style.paddingRight = "".concat(states.previousBodyPadding + measureScrollbar(), "px");
    }
  };
  const undoScrollbar = () => {
    if (states.previousBodyPadding !== null) {
      document.body.style.paddingRight = "".concat(states.previousBodyPadding, "px");
      states.previousBodyPadding = null;
    }
  };

  /* istanbul ignore file */

  const iOSfix = () => {
    const iOS = /iPad|iPhone|iPod/.test(navigator.userAgent) && !window.MSStream || navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1;

    if (iOS && !hasClass(document.body, swalClasses.iosfix)) {
      const offset = document.body.scrollTop;
      document.body.style.top = "".concat(offset * -1, "px");
      addClass(document.body, swalClasses.iosfix);
      lockBodyScroll();
      addBottomPaddingForTallPopups(); // #1948
    }
  };

  const addBottomPaddingForTallPopups = () => {
    const safari = !navigator.userAgent.match(/(CriOS|FxiOS|EdgiOS|YaBrowser|UCBrowser)/i);

    if (safari) {
      const bottomPanelHeight = 44;

      if (getPopup().scrollHeight > window.innerHeight - bottomPanelHeight) {
        getContainer().style.paddingBottom = "".concat(bottomPanelHeight, "px");
      }
    }
  };

  const lockBodyScroll = () => {
    // #1246
    const container = getContainer();
    let preventTouchMove;

    container.ontouchstart = e => {
      preventTouchMove = shouldPreventTouchMove(e);
    };

    container.ontouchmove = e => {
      if (preventTouchMove) {
        e.preventDefault();
        e.stopPropagation();
      }
    };
  };

  const shouldPreventTouchMove = event => {
    const target = event.target;
    const container = getContainer();

    if (isStylys(event) || isZoom(event)) {
      return false;
    }

    if (target === container) {
      return true;
    }

    if (!isScrollable(container) && target.tagName !== 'INPUT' && // #1603
    target.tagName !== 'TEXTAREA' && // #2266
    !(isScrollable(getHtmlContainer()) && // #1944
    getHtmlContainer().contains(target))) {
      return true;
    }

    return false;
  };

  const isStylys = event => {
    // #1786
    return event.touches && event.touches.length && event.touches[0].touchType === 'stylus';
  };

  const isZoom = event => {
    // #1891
    return event.touches && event.touches.length > 1;
  };

  const undoIOSfix = () => {
    if (hasClass(document.body, swalClasses.iosfix)) {
      const offset = parseInt(document.body.style.top, 10);
      removeClass(document.body, swalClasses.iosfix);
      document.body.style.top = '';
      document.body.scrollTop = offset * -1;
    }
  };

  // Adding aria-hidden="true" to elements outside of the active modal dialog ensures that
  // elements not within the active modal dialog will not be surfaced if a user opens a screen
  // reader’s list of elements (headings, form controls, landmarks, etc.) in the document.

  const setAriaHidden = () => {
    const bodyChildren = toArray(document.body.children);
    bodyChildren.forEach(el => {
      if (el === getContainer() || el.contains(getContainer())) {
        return;
      }

      if (el.hasAttribute('aria-hidden')) {
        el.setAttribute('data-previous-aria-hidden', el.getAttribute('aria-hidden'));
      }

      el.setAttribute('aria-hidden', 'true');
    });
  };
  const unsetAriaHidden = () => {
    const bodyChildren = toArray(document.body.children);
    bodyChildren.forEach(el => {
      if (el.hasAttribute('data-previous-aria-hidden')) {
        el.setAttribute('aria-hidden', el.getAttribute('data-previous-aria-hidden'));
        el.removeAttribute('data-previous-aria-hidden');
      } else {
        el.removeAttribute('aria-hidden');
      }
    });
  };

  /**
   * This module contains `WeakMap`s for each effectively-"private  property" that a `Swal` has.
   * For example, to set the private property "foo" of `this` to "bar", you can `privateProps.foo.set(this, 'bar')`
   * This is the approach that Babel will probably take to implement private methods/fields
   *   https://github.com/tc39/proposal-private-methods
   *   https://github.com/babel/babel/pull/7555
   * Once we have the changes from that PR in Babel, and our core class fits reasonable in *one module*
   *   then we can use that language feature.
   */
  var privateMethods = {
    swalPromiseResolve: new WeakMap(),
    swalPromiseReject: new WeakMap()
  };

  /*
   * Instance method to close sweetAlert
   */

  function removePopupAndResetState(instance, container, returnFocus, didClose) {
    if (isToast()) {
      triggerDidCloseAndDispose(instance, didClose);
    } else {
      restoreActiveElement(returnFocus).then(() => triggerDidCloseAndDispose(instance, didClose));
      globalState.keydownTarget.removeEventListener('keydown', globalState.keydownHandler, {
        capture: globalState.keydownListenerCapture
      });
      globalState.keydownHandlerAdded = false;
    }

    const isSafari = /^((?!chrome|android).)*safari/i.test(navigator.userAgent); // workaround for #2088
    // for some reason removing the container in Safari will scroll the document to bottom

    if (isSafari) {
      container.setAttribute('style', 'display:none !important');
      container.removeAttribute('class');
      container.innerHTML = '';
    } else {
      container.remove();
    }

    if (isModal()) {
      undoScrollbar();
      undoIOSfix();
      unsetAriaHidden();
    }

    removeBodyClasses();
  }

  function removeBodyClasses() {
    removeClass([document.documentElement, document.body], [swalClasses.shown, swalClasses['height-auto'], swalClasses['no-backdrop'], swalClasses['toast-shown']]);
  }

  function close(resolveValue) {
    resolveValue = prepareResolveValue(resolveValue);
    const swalPromiseResolve = privateMethods.swalPromiseResolve.get(this);
    const didClose = triggerClosePopup(this);

    if (this.isAwaitingPromise()) {
      // A swal awaiting for a promise (after a click on Confirm or Deny) cannot be dismissed anymore #2335
      if (!resolveValue.isDismissed) {
        handleAwaitingPromise(this);
        swalPromiseResolve(resolveValue);
      }
    } else if (didClose) {
      // Resolve Swal promise
      swalPromiseResolve(resolveValue);
    }
  }
  function isAwaitingPromise() {
    return !!privateProps.awaitingPromise.get(this);
  }

  const triggerClosePopup = instance => {
    const popup = getPopup();

    if (!popup) {
      return false;
    }

    const innerParams = privateProps.innerParams.get(instance);

    if (!innerParams || hasClass(popup, innerParams.hideClass.popup)) {
      return false;
    }

    removeClass(popup, innerParams.showClass.popup);
    addClass(popup, innerParams.hideClass.popup);
    const backdrop = getContainer();
    removeClass(backdrop, innerParams.showClass.backdrop);
    addClass(backdrop, innerParams.hideClass.backdrop);
    handlePopupAnimation(instance, popup, innerParams);
    return true;
  };

  function rejectPromise(error) {
    const rejectPromise = privateMethods.swalPromiseReject.get(this);
    handleAwaitingPromise(this);

    if (rejectPromise) {
      // Reject Swal promise
      rejectPromise(error);
    }
  }

  const handleAwaitingPromise = instance => {
    if (instance.isAwaitingPromise()) {
      privateProps.awaitingPromise.delete(instance); // The instance might have been previously partly destroyed, we must resume the destroy process in this case #2335

      if (!privateProps.innerParams.get(instance)) {
        instance._destroy();
      }
    }
  };

  const prepareResolveValue = resolveValue => {
    // When user calls Swal.close()
    if (typeof resolveValue === 'undefined') {
      return {
        isConfirmed: false,
        isDenied: false,
        isDismissed: true
      };
    }

    return Object.assign({
      isConfirmed: false,
      isDenied: false,
      isDismissed: false
    }, resolveValue);
  };

  const handlePopupAnimation = (instance, popup, innerParams) => {
    const container = getContainer(); // If animation is supported, animate

    const animationIsSupported = animationEndEvent && hasCssAnimation(popup);

    if (typeof innerParams.willClose === 'function') {
      innerParams.willClose(popup);
    }

    if (animationIsSupported) {
      animatePopup(instance, popup, container, innerParams.returnFocus, innerParams.didClose);
    } else {
      // Otherwise, remove immediately
      removePopupAndResetState(instance, container, innerParams.returnFocus, innerParams.didClose);
    }
  };

  const animatePopup = (instance, popup, container, returnFocus, didClose) => {
    globalState.swalCloseEventFinishedCallback = removePopupAndResetState.bind(null, instance, container, returnFocus, didClose);
    popup.addEventListener(animationEndEvent, function (e) {
      if (e.target === popup) {
        globalState.swalCloseEventFinishedCallback();
        delete globalState.swalCloseEventFinishedCallback;
      }
    });
  };

  const triggerDidCloseAndDispose = (instance, didClose) => {
    setTimeout(() => {
      if (typeof didClose === 'function') {
        didClose.bind(instance.params)();
      }

      instance._destroy();
    });
  };

  function setButtonsDisabled(instance, buttons, disabled) {
    const domCache = privateProps.domCache.get(instance);
    buttons.forEach(button => {
      domCache[button].disabled = disabled;
    });
  }

  function setInputDisabled(input, disabled) {
    if (!input) {
      return false;
    }

    if (input.type === 'radio') {
      const radiosContainer = input.parentNode.parentNode;
      const radios = radiosContainer.querySelectorAll('input');

      for (let i = 0; i < radios.length; i++) {
        radios[i].disabled = disabled;
      }
    } else {
      input.disabled = disabled;
    }
  }

  function enableButtons() {
    setButtonsDisabled(this, ['confirmButton', 'denyButton', 'cancelButton'], false);
  }
  function disableButtons() {
    setButtonsDisabled(this, ['confirmButton', 'denyButton', 'cancelButton'], true);
  }
  function enableInput() {
    return setInputDisabled(this.getInput(), false);
  }
  function disableInput() {
    return setInputDisabled(this.getInput(), true);
  }

  function showValidationMessage(error) {
    const domCache = privateProps.domCache.get(this);
    const params = privateProps.innerParams.get(this);
    setInnerHtml(domCache.validationMessage, error);
    domCache.validationMessage.className = swalClasses['validation-message'];

    if (params.customClass && params.customClass.validationMessage) {
      addClass(domCache.validationMessage, params.customClass.validationMessage);
    }

    show(domCache.validationMessage);
    const input = this.getInput();

    if (input) {
      input.setAttribute('aria-invalid', true);
      input.setAttribute('aria-describedby', swalClasses['validation-message']);
      focusInput(input);
      addClass(input, swalClasses.inputerror);
    }
  } // Hide block with validation message

  function resetValidationMessage$1() {
    const domCache = privateProps.domCache.get(this);

    if (domCache.validationMessage) {
      hide(domCache.validationMessage);
    }

    const input = this.getInput();

    if (input) {
      input.removeAttribute('aria-invalid');
      input.removeAttribute('aria-describedby');
      removeClass(input, swalClasses.inputerror);
    }
  }

  function getProgressSteps$1() {
    const domCache = privateProps.domCache.get(this);
    return domCache.progressSteps;
  }

  class Timer {
    constructor(callback, delay) {
      this.callback = callback;
      this.remaining = delay;
      this.running = false;
      this.start();
    }

    start() {
      if (!this.running) {
        this.running = true;
        this.started = new Date();
        this.id = setTimeout(this.callback, this.remaining);
      }

      return this.remaining;
    }

    stop() {
      if (this.running) {
        this.running = false;
        clearTimeout(this.id);
        this.remaining -= new Date() - this.started;
      }

      return this.remaining;
    }

    increase(n) {
      const running = this.running;

      if (running) {
        this.stop();
      }

      this.remaining += n;

      if (running) {
        this.start();
      }

      return this.remaining;
    }

    getTimerLeft() {
      if (this.running) {
        this.stop();
        this.start();
      }

      return this.remaining;
    }

    isRunning() {
      return this.running;
    }

  }

  var defaultInputValidators = {
    email: (string, validationMessage) => {
      return /^[a-zA-Z0-9.+_-]+@[a-zA-Z0-9.-]+\.[a-zA-Z0-9-]{2,24}$/.test(string) ? Promise.resolve() : Promise.resolve(validationMessage || 'Invalid email address');
    },
    url: (string, validationMessage) => {
      // taken from https://stackoverflow.com/a/3809435 with a small change from #1306 and #2013
      return /^https?:\/\/(www\.)?[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-z]{2,63}\b([-a-zA-Z0-9@:%_+.~#?&/=]*)$/.test(string) ? Promise.resolve() : Promise.resolve(validationMessage || 'Invalid URL');
    }
  };

  function setDefaultInputValidators(params) {
    // Use default `inputValidator` for supported input types if not provided
    if (!params.inputValidator) {
      Object.keys(defaultInputValidators).forEach(key => {
        if (params.input === key) {
          params.inputValidator = defaultInputValidators[key];
        }
      });
    }
  }

  function validateCustomTargetElement(params) {
    // Determine if the custom target element is valid
    if (!params.target || typeof params.target === 'string' && !document.querySelector(params.target) || typeof params.target !== 'string' && !params.target.appendChild) {
      warn('Target parameter is not valid, defaulting to "body"');
      params.target = 'body';
    }
  }
  /**
   * Set type, text and actions on popup
   *
   * @param params
   * @returns {boolean}
   */


  function setParameters(params) {
    setDefaultInputValidators(params); // showLoaderOnConfirm && preConfirm

    if (params.showLoaderOnConfirm && !params.preConfirm) {
      warn('showLoaderOnConfirm is set to true, but preConfirm is not defined.\n' + 'showLoaderOnConfirm should be used together with preConfirm, see usage example:\n' + 'https://sweetalert2.github.io/#ajax-request');
    }

    validateCustomTargetElement(params); // Replace newlines with <br> in title

    if (typeof params.title === 'string') {
      params.title = params.title.split('\n').join('<br />');
    }

    init(params);
  }

  const swalStringParams = ['swal-title', 'swal-html', 'swal-footer'];
  const getTemplateParams = params => {
    const template = typeof params.template === 'string' ? document.querySelector(params.template) : params.template;

    if (!template) {
      return {};
    }

    const templateContent = template.content;
    showWarningsForElements(templateContent);
    const result = Object.assign(getSwalParams(templateContent), getSwalButtons(templateContent), getSwalImage(templateContent), getSwalIcon(templateContent), getSwalInput(templateContent), getSwalStringParams(templateContent, swalStringParams));
    return result;
  };

  const getSwalParams = templateContent => {
    const result = {};
    toArray(templateContent.querySelectorAll('swal-param')).forEach(param => {
      showWarningsForAttributes(param, ['name', 'value']);
      const paramName = param.getAttribute('name');
      let value = param.getAttribute('value');

      if (typeof defaultParams[paramName] === 'boolean' && value === 'false') {
        value = false;
      }

      if (typeof defaultParams[paramName] === 'object') {
        value = JSON.parse(value);
      }

      result[paramName] = value;
    });
    return result;
  };

  const getSwalButtons = templateContent => {
    const result = {};
    toArray(templateContent.querySelectorAll('swal-button')).forEach(button => {
      showWarningsForAttributes(button, ['type', 'color', 'aria-label']);
      const type = button.getAttribute('type');
      result["".concat(type, "ButtonText")] = button.innerHTML;
      result["show".concat(capitalizeFirstLetter(type), "Button")] = true;

      if (button.hasAttribute('color')) {
        result["".concat(type, "ButtonColor")] = button.getAttribute('color');
      }

      if (button.hasAttribute('aria-label')) {
        result["".concat(type, "ButtonAriaLabel")] = button.getAttribute('aria-label');
      }
    });
    return result;
  };

  const getSwalImage = templateContent => {
    const result = {};
    const image = templateContent.querySelector('swal-image');

    if (image) {
      showWarningsForAttributes(image, ['src', 'width', 'height', 'alt']);

      if (image.hasAttribute('src')) {
        result.imageUrl = image.getAttribute('src');
      }

      if (image.hasAttribute('width')) {
        result.imageWidth = image.getAttribute('width');
      }

      if (image.hasAttribute('height')) {
        result.imageHeight = image.getAttribute('height');
      }

      if (image.hasAttribute('alt')) {
        result.imageAlt = image.getAttribute('alt');
      }
    }

    return result;
  };

  const getSwalIcon = templateContent => {
    const result = {};
    const icon = templateContent.querySelector('swal-icon');

    if (icon) {
      showWarningsForAttributes(icon, ['type', 'color']);

      if (icon.hasAttribute('type')) {
        result.icon = icon.getAttribute('type');
      }

      if (icon.hasAttribute('color')) {
        result.iconColor = icon.getAttribute('color');
      }

      result.iconHtml = icon.innerHTML;
    }

    return result;
  };

  const getSwalInput = templateContent => {
    const result = {};
    const input = templateContent.querySelector('swal-input');

    if (input) {
      showWarningsForAttributes(input, ['type', 'label', 'placeholder', 'value']);
      result.input = input.getAttribute('type') || 'text';

      if (input.hasAttribute('label')) {
        result.inputLabel = input.getAttribute('label');
      }

      if (input.hasAttribute('placeholder')) {
        result.inputPlaceholder = input.getAttribute('placeholder');
      }

      if (input.hasAttribute('value')) {
        result.inputValue = input.getAttribute('value');
      }
    }

    const inputOptions = templateContent.querySelectorAll('swal-input-option');

    if (inputOptions.length) {
      result.inputOptions = {};
      toArray(inputOptions).forEach(option => {
        showWarningsForAttributes(option, ['value']);
        const optionValue = option.getAttribute('value');
        const optionName = option.innerHTML;
        result.inputOptions[optionValue] = optionName;
      });
    }

    return result;
  };

  const getSwalStringParams = (templateContent, paramNames) => {
    const result = {};

    for (const i in paramNames) {
      const paramName = paramNames[i];
      const tag = templateContent.querySelector(paramName);

      if (tag) {
        showWarningsForAttributes(tag, []);
        result[paramName.replace(/^swal-/, '')] = tag.innerHTML.trim();
      }
    }

    return result;
  };

  const showWarningsForElements = template => {
    const allowedElements = swalStringParams.concat(['swal-param', 'swal-button', 'swal-image', 'swal-icon', 'swal-input', 'swal-input-option']);
    toArray(template.children).forEach(el => {
      const tagName = el.tagName.toLowerCase();

      if (allowedElements.indexOf(tagName) === -1) {
        warn("Unrecognized element <".concat(tagName, ">"));
      }
    });
  };

  const showWarningsForAttributes = (el, allowedAttributes) => {
    toArray(el.attributes).forEach(attribute => {
      if (allowedAttributes.indexOf(attribute.name) === -1) {
        warn(["Unrecognized attribute \"".concat(attribute.name, "\" on <").concat(el.tagName.toLowerCase(), ">."), "".concat(allowedAttributes.length ? "Allowed attributes are: ".concat(allowedAttributes.join(', ')) : 'To set the value, use HTML within the element.')]);
      }
    });
  };

  const SHOW_CLASS_TIMEOUT = 10;
  /**
   * Open popup, add necessary classes and styles, fix scrollbar
   *
   * @param params
   */

  const openPopup = params => {
    const container = getContainer();
    const popup = getPopup();

    if (typeof params.willOpen === 'function') {
      params.willOpen(popup);
    }

    const bodyStyles = window.getComputedStyle(document.body);
    const initialBodyOverflow = bodyStyles.overflowY;
    addClasses$1(container, popup, params); // scrolling is 'hidden' until animation is done, after that 'auto'

    setTimeout(() => {
      setScrollingVisibility(container, popup);
    }, SHOW_CLASS_TIMEOUT);

    if (isModal()) {
      fixScrollContainer(container, params.scrollbarPadding, initialBodyOverflow);
      setAriaHidden();
    }

    if (!isToast() && !globalState.previousActiveElement) {
      globalState.previousActiveElement = document.activeElement;
    }

    if (typeof params.didOpen === 'function') {
      setTimeout(() => params.didOpen(popup));
    }

    removeClass(container, swalClasses['no-transition']);
  };

  const swalOpenAnimationFinished = event => {
    const popup = getPopup();

    if (event.target !== popup) {
      return;
    }

    const container = getContainer();
    popup.removeEventListener(animationEndEvent, swalOpenAnimationFinished);
    container.style.overflowY = 'auto';
  };

  const setScrollingVisibility = (container, popup) => {
    if (animationEndEvent && hasCssAnimation(popup)) {
      container.style.overflowY = 'hidden';
      popup.addEventListener(animationEndEvent, swalOpenAnimationFinished);
    } else {
      container.style.overflowY = 'auto';
    }
  };

  const fixScrollContainer = (container, scrollbarPadding, initialBodyOverflow) => {
    iOSfix();

    if (scrollbarPadding && initialBodyOverflow !== 'hidden') {
      fixScrollbar();
    } // sweetalert2/issues/1247


    setTimeout(() => {
      container.scrollTop = 0;
    });
  };

  const addClasses$1 = (container, popup, params) => {
    addClass(container, params.showClass.backdrop); // the workaround with setting/unsetting opacity is needed for #2019 and 2059

    popup.style.setProperty('opacity', '0', 'important');
    show(popup, 'grid');
    setTimeout(() => {
      // Animate popup right after showing it
      addClass(popup, params.showClass.popup); // and remove the opacity workaround

      popup.style.removeProperty('opacity');
    }, SHOW_CLASS_TIMEOUT); // 10ms in order to fix #2062

    addClass([document.documentElement, document.body], swalClasses.shown);

    if (params.heightAuto && params.backdrop && !params.toast) {
      addClass([document.documentElement, document.body], swalClasses['height-auto']);
    }
  };

  const handleInputOptionsAndValue = (instance, params) => {
    if (params.input === 'select' || params.input === 'radio') {
      handleInputOptions(instance, params);
    } else if (['text', 'email', 'number', 'tel', 'textarea'].includes(params.input) && (hasToPromiseFn(params.inputValue) || isPromise(params.inputValue))) {
      showLoading(getConfirmButton());
      handleInputValue(instance, params);
    }
  };
  const getInputValue = (instance, innerParams) => {
    const input = instance.getInput();

    if (!input) {
      return null;
    }

    switch (innerParams.input) {
      case 'checkbox':
        return getCheckboxValue(input);

      case 'radio':
        return getRadioValue(input);

      case 'file':
        return getFileValue(input);

      default:
        return innerParams.inputAutoTrim ? input.value.trim() : input.value;
    }
  };

  const getCheckboxValue = input => input.checked ? 1 : 0;

  const getRadioValue = input => input.checked ? input.value : null;

  const getFileValue = input => input.files.length ? input.getAttribute('multiple') !== null ? input.files : input.files[0] : null;

  const handleInputOptions = (instance, params) => {
    const popup = getPopup();

    const processInputOptions = inputOptions => populateInputOptions[params.input](popup, formatInputOptions(inputOptions), params);

    if (hasToPromiseFn(params.inputOptions) || isPromise(params.inputOptions)) {
      showLoading(getConfirmButton());
      asPromise(params.inputOptions).then(inputOptions => {
        instance.hideLoading();
        processInputOptions(inputOptions);
      });
    } else if (typeof params.inputOptions === 'object') {
      processInputOptions(params.inputOptions);
    } else {
      error("Unexpected type of inputOptions! Expected object, Map or Promise, got ".concat(typeof params.inputOptions));
    }
  };

  const handleInputValue = (instance, params) => {
    const input = instance.getInput();
    hide(input);
    asPromise(params.inputValue).then(inputValue => {
      input.value = params.input === 'number' ? parseFloat(inputValue) || 0 : "".concat(inputValue);
      show(input);
      input.focus();
      instance.hideLoading();
    }).catch(err => {
      error("Error in inputValue promise: ".concat(err));
      input.value = '';
      show(input);
      input.focus();
      instance.hideLoading();
    });
  };

  const populateInputOptions = {
    select: (popup, inputOptions, params) => {
      const select = getChildByClass(popup, swalClasses.select);

      const renderOption = (parent, optionLabel, optionValue) => {
        const option = document.createElement('option');
        option.value = optionValue;
        setInnerHtml(option, optionLabel);
        option.selected = isSelected(optionValue, params.inputValue);
        parent.appendChild(option);
      };

      inputOptions.forEach(inputOption => {
        const optionValue = inputOption[0];
        const optionLabel = inputOption[1]; // <optgroup> spec:
        // https://www.w3.org/TR/html401/interact/forms.html#h-17.6
        // "...all OPTGROUP elements must be specified directly within a SELECT element (i.e., groups may not be nested)..."
        // check whether this is a <optgroup>

        if (Array.isArray(optionLabel)) {
          // if it is an array, then it is an <optgroup>
          const optgroup = document.createElement('optgroup');
          optgroup.label = optionValue;
          optgroup.disabled = false; // not configurable for now

          select.appendChild(optgroup);
          optionLabel.forEach(o => renderOption(optgroup, o[1], o[0]));
        } else {
          // case of <option>
          renderOption(select, optionLabel, optionValue);
        }
      });
      select.focus();
    },
    radio: (popup, inputOptions, params) => {
      const radio = getChildByClass(popup, swalClasses.radio);
      inputOptions.forEach(inputOption => {
        const radioValue = inputOption[0];
        const radioLabel = inputOption[1];
        const radioInput = document.createElement('input');
        const radioLabelElement = document.createElement('label');
        radioInput.type = 'radio';
        radioInput.name = swalClasses.radio;
        radioInput.value = radioValue;

        if (isSelected(radioValue, params.inputValue)) {
          radioInput.checked = true;
        }

        const label = document.createElement('span');
        setInnerHtml(label, radioLabel);
        label.className = swalClasses.label;
        radioLabelElement.appendChild(radioInput);
        radioLabelElement.appendChild(label);
        radio.appendChild(radioLabelElement);
      });
      const radios = radio.querySelectorAll('input');

      if (radios.length) {
        radios[0].focus();
      }
    }
  };
  /**
   * Converts `inputOptions` into an array of `[value, label]`s
   * @param inputOptions
   */

  const formatInputOptions = inputOptions => {
    const result = [];

    if (typeof Map !== 'undefined' && inputOptions instanceof Map) {
      inputOptions.forEach((value, key) => {
        let valueFormatted = value;

        if (typeof valueFormatted === 'object') {
          // case of <optgroup>
          valueFormatted = formatInputOptions(valueFormatted);
        }

        result.push([key, valueFormatted]);
      });
    } else {
      Object.keys(inputOptions).forEach(key => {
        let valueFormatted = inputOptions[key];

        if (typeof valueFormatted === 'object') {
          // case of <optgroup>
          valueFormatted = formatInputOptions(valueFormatted);
        }

        result.push([key, valueFormatted]);
      });
    }

    return result;
  };

  const isSelected = (optionValue, inputValue) => {
    return inputValue && inputValue.toString() === optionValue.toString();
  };

  const handleConfirmButtonClick = instance => {
    const innerParams = privateProps.innerParams.get(instance);
    instance.disableButtons();

    if (innerParams.input) {
      handleConfirmOrDenyWithInput(instance, 'confirm');
    } else {
      confirm(instance, true);
    }
  };
  const handleDenyButtonClick = instance => {
    const innerParams = privateProps.innerParams.get(instance);
    instance.disableButtons();

    if (innerParams.returnInputValueOnDeny) {
      handleConfirmOrDenyWithInput(instance, 'deny');
    } else {
      deny(instance, false);
    }
  };
  const handleCancelButtonClick = (instance, dismissWith) => {
    instance.disableButtons();
    dismissWith(DismissReason.cancel);
  };

  const handleConfirmOrDenyWithInput = (instance, type
  /* 'confirm' | 'deny' */
  ) => {
    const innerParams = privateProps.innerParams.get(instance);
    const inputValue = getInputValue(instance, innerParams);

    if (innerParams.inputValidator) {
      handleInputValidator(instance, inputValue, type);
    } else if (!instance.getInput().checkValidity()) {
      instance.enableButtons();
      instance.showValidationMessage(innerParams.validationMessage);
    } else if (type === 'deny') {
      deny(instance, inputValue);
    } else {
      confirm(instance, inputValue);
    }
  };

  const handleInputValidator = (instance, inputValue, type
  /* 'confirm' | 'deny' */
  ) => {
    const innerParams = privateProps.innerParams.get(instance);
    instance.disableInput();
    const validationPromise = Promise.resolve().then(() => asPromise(innerParams.inputValidator(inputValue, innerParams.validationMessage)));
    validationPromise.then(validationMessage => {
      instance.enableButtons();
      instance.enableInput();

      if (validationMessage) {
        instance.showValidationMessage(validationMessage);
      } else if (type === 'deny') {
        deny(instance, inputValue);
      } else {
        confirm(instance, inputValue);
      }
    });
  };

  const deny = (instance, value) => {
    const innerParams = privateProps.innerParams.get(instance || undefined);

    if (innerParams.showLoaderOnDeny) {
      showLoading(getDenyButton());
    }

    if (innerParams.preDeny) {
      privateProps.awaitingPromise.set(instance || undefined, true); // Flagging the instance as awaiting a promise so it's own promise's reject/resolve methods doesnt get destroyed until the result from this preDeny's promise is received

      const preDenyPromise = Promise.resolve().then(() => asPromise(innerParams.preDeny(value, innerParams.validationMessage)));
      preDenyPromise.then(preDenyValue => {
        if (preDenyValue === false) {
          instance.hideLoading();
        } else {
          instance.closePopup({
            isDenied: true,
            value: typeof preDenyValue === 'undefined' ? value : preDenyValue
          });
        }
      }).catch(error$$1 => rejectWith(instance || undefined, error$$1));
    } else {
      instance.closePopup({
        isDenied: true,
        value
      });
    }
  };

  const succeedWith = (instance, value) => {
    instance.closePopup({
      isConfirmed: true,
      value
    });
  };

  const rejectWith = (instance, error$$1) => {
    instance.rejectPromise(error$$1);
  };

  const confirm = (instance, value) => {
    const innerParams = privateProps.innerParams.get(instance || undefined);

    if (innerParams.showLoaderOnConfirm) {
      showLoading();
    }

    if (innerParams.preConfirm) {
      instance.resetValidationMessage();
      privateProps.awaitingPromise.set(instance || undefined, true); // Flagging the instance as awaiting a promise so it's own promise's reject/resolve methods doesnt get destroyed until the result from this preConfirm's promise is received

      const preConfirmPromise = Promise.resolve().then(() => asPromise(innerParams.preConfirm(value, innerParams.validationMessage)));
      preConfirmPromise.then(preConfirmValue => {
        if (isVisible(getValidationMessage()) || preConfirmValue === false) {
          instance.hideLoading();
        } else {
          succeedWith(instance, typeof preConfirmValue === 'undefined' ? value : preConfirmValue);
        }
      }).catch(error$$1 => rejectWith(instance || undefined, error$$1));
    } else {
      succeedWith(instance, value);
    }
  };

  const addKeydownHandler = (instance, globalState, innerParams, dismissWith) => {
    if (globalState.keydownTarget && globalState.keydownHandlerAdded) {
      globalState.keydownTarget.removeEventListener('keydown', globalState.keydownHandler, {
        capture: globalState.keydownListenerCapture
      });
      globalState.keydownHandlerAdded = false;
    }

    if (!innerParams.toast) {
      globalState.keydownHandler = e => keydownHandler(instance, e, dismissWith);

      globalState.keydownTarget = innerParams.keydownListenerCapture ? window : getPopup();
      globalState.keydownListenerCapture = innerParams.keydownListenerCapture;
      globalState.keydownTarget.addEventListener('keydown', globalState.keydownHandler, {
        capture: globalState.keydownListenerCapture
      });
      globalState.keydownHandlerAdded = true;
    }
  }; // Focus handling

  const setFocus = (innerParams, index, increment) => {
    const focusableElements = getFocusableElements(); // search for visible elements and select the next possible match

    if (focusableElements.length) {
      index = index + increment; // rollover to first item

      if (index === focusableElements.length) {
        index = 0; // go to last item
      } else if (index === -1) {
        index = focusableElements.length - 1;
      }

      return focusableElements[index].focus();
    } // no visible focusable elements, focus the popup


    getPopup().focus();
  };
  const arrowKeysNextButton = ['ArrowRight', 'ArrowDown'];
  const arrowKeysPreviousButton = ['ArrowLeft', 'ArrowUp'];

  const keydownHandler = (instance, e, dismissWith) => {
    const innerParams = privateProps.innerParams.get(instance);

    if (!innerParams) {
      return; // This instance has already been destroyed
    }

    if (innerParams.stopKeydownPropagation) {
      e.stopPropagation();
    } // ENTER


    if (e.key === 'Enter') {
      handleEnter(instance, e, innerParams); // TAB
    } else if (e.key === 'Tab') {
      handleTab(e, innerParams); // ARROWS - switch focus between buttons
    } else if ([...arrowKeysNextButton, ...arrowKeysPreviousButton].includes(e.key)) {
      handleArrows(e.key); // ESC
    } else if (e.key === 'Escape') {
      handleEsc(e, innerParams, dismissWith);
    }
  };

  const handleEnter = (instance, e, innerParams) => {
    // #720 #721
    if (e.isComposing) {
      return;
    }

    if (e.target && instance.getInput() && e.target.outerHTML === instance.getInput().outerHTML) {
      if (['textarea', 'file'].includes(innerParams.input)) {
        return; // do not submit
      }

      clickConfirm();
      e.preventDefault();
    }
  };

  const handleTab = (e, innerParams) => {
    const targetElement = e.target;
    const focusableElements = getFocusableElements();
    let btnIndex = -1;

    for (let i = 0; i < focusableElements.length; i++) {
      if (targetElement === focusableElements[i]) {
        btnIndex = i;
        break;
      }
    }

    if (!e.shiftKey) {
      // Cycle to the next button
      setFocus(innerParams, btnIndex, 1);
    } else {
      // Cycle to the prev button
      setFocus(innerParams, btnIndex, -1);
    }

    e.stopPropagation();
    e.preventDefault();
  };

  const handleArrows = key => {
    const confirmButton = getConfirmButton();
    const denyButton = getDenyButton();
    const cancelButton = getCancelButton();

    if (![confirmButton, denyButton, cancelButton].includes(document.activeElement)) {
      return;
    }

    const sibling = arrowKeysNextButton.includes(key) ? 'nextElementSibling' : 'previousElementSibling';
    const buttonToFocus = document.activeElement[sibling];

    if (buttonToFocus) {
      buttonToFocus.focus();
    }
  };

  const handleEsc = (e, innerParams, dismissWith) => {
    if (callIfFunction(innerParams.allowEscapeKey)) {
      e.preventDefault();
      dismissWith(DismissReason.esc);
    }
  };

  const handlePopupClick = (instance, domCache, dismissWith) => {
    const innerParams = privateProps.innerParams.get(instance);

    if (innerParams.toast) {
      handleToastClick(instance, domCache, dismissWith);
    } else {
      // Ignore click events that had mousedown on the popup but mouseup on the container
      // This can happen when the user drags a slider
      handleModalMousedown(domCache); // Ignore click events that had mousedown on the container but mouseup on the popup

      handleContainerMousedown(domCache);
      handleModalClick(instance, domCache, dismissWith);
    }
  };

  const handleToastClick = (instance, domCache, dismissWith) => {
    // Closing toast by internal click
    domCache.popup.onclick = () => {
      const innerParams = privateProps.innerParams.get(instance);

      if (innerParams.showConfirmButton || innerParams.showDenyButton || innerParams.showCancelButton || innerParams.showCloseButton || innerParams.timer || innerParams.input) {
        return;
      }

      dismissWith(DismissReason.close);
    };
  };

  let ignoreOutsideClick = false;

  const handleModalMousedown = domCache => {
    domCache.popup.onmousedown = () => {
      domCache.container.onmouseup = function (e) {
        domCache.container.onmouseup = undefined; // We only check if the mouseup target is the container because usually it doesn't
        // have any other direct children aside of the popup

        if (e.target === domCache.container) {
          ignoreOutsideClick = true;
        }
      };
    };
  };

  const handleContainerMousedown = domCache => {
    domCache.container.onmousedown = () => {
      domCache.popup.onmouseup = function (e) {
        domCache.popup.onmouseup = undefined; // We also need to check if the mouseup target is a child of the popup

        if (e.target === domCache.popup || domCache.popup.contains(e.target)) {
          ignoreOutsideClick = true;
        }
      };
    };
  };

  const handleModalClick = (instance, domCache, dismissWith) => {
    domCache.container.onclick = e => {
      const innerParams = privateProps.innerParams.get(instance);

      if (ignoreOutsideClick) {
        ignoreOutsideClick = false;
        return;
      }

      if (e.target === domCache.container && callIfFunction(innerParams.allowOutsideClick)) {
        dismissWith(DismissReason.backdrop);
      }
    };
  };

  function _main(userParams) {
    let mixinParams = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : {};
    showWarningsForParams(Object.assign({}, mixinParams, userParams));

    if (globalState.currentInstance) {
      globalState.currentInstance._destroy();

      if (isModal()) {
        unsetAriaHidden();
      }
    }

    globalState.currentInstance = this;
    const innerParams = prepareParams(userParams, mixinParams);
    setParameters(innerParams);
    Object.freeze(innerParams); // clear the previous timer

    if (globalState.timeout) {
      globalState.timeout.stop();
      delete globalState.timeout;
    } // clear the restore focus timeout


    clearTimeout(globalState.restoreFocusTimeout);
    const domCache = populateDomCache(this);
    render(this, innerParams);
    privateProps.innerParams.set(this, innerParams);
    return swalPromise(this, domCache, innerParams);
  }

  const prepareParams = (userParams, mixinParams) => {
    const templateParams = getTemplateParams(userParams);
    const params = Object.assign({}, defaultParams, mixinParams, templateParams, userParams); // precedence is described in #2131

    params.showClass = Object.assign({}, defaultParams.showClass, params.showClass);
    params.hideClass = Object.assign({}, defaultParams.hideClass, params.hideClass);
    return params;
  };

  const swalPromise = (instance, domCache, innerParams) => {
    return new Promise((resolve, reject) => {
      // functions to handle all closings/dismissals
      const dismissWith = dismiss => {
        instance.closePopup({
          isDismissed: true,
          dismiss
        });
      };

      privateMethods.swalPromiseResolve.set(instance, resolve);
      privateMethods.swalPromiseReject.set(instance, reject);

      domCache.confirmButton.onclick = () => handleConfirmButtonClick(instance);

      domCache.denyButton.onclick = () => handleDenyButtonClick(instance);

      domCache.cancelButton.onclick = () => handleCancelButtonClick(instance, dismissWith);

      domCache.closeButton.onclick = () => dismissWith(DismissReason.close);

      handlePopupClick(instance, domCache, dismissWith);
      addKeydownHandler(instance, globalState, innerParams, dismissWith);
      handleInputOptionsAndValue(instance, innerParams);
      openPopup(innerParams);
      setupTimer(globalState, innerParams, dismissWith);
      initFocus(domCache, innerParams); // Scroll container to top on open (#1247, #1946)

      setTimeout(() => {
        domCache.container.scrollTop = 0;
      });
    });
  };

  const populateDomCache = instance => {
    const domCache = {
      popup: getPopup(),
      container: getContainer(),
      actions: getActions(),
      confirmButton: getConfirmButton(),
      denyButton: getDenyButton(),
      cancelButton: getCancelButton(),
      loader: getLoader(),
      closeButton: getCloseButton(),
      validationMessage: getValidationMessage(),
      progressSteps: getProgressSteps()
    };
    privateProps.domCache.set(instance, domCache);
    return domCache;
  };

  const setupTimer = (globalState$$1, innerParams, dismissWith) => {
    const timerProgressBar = getTimerProgressBar();
    hide(timerProgressBar);

    if (innerParams.timer) {
      globalState$$1.timeout = new Timer(() => {
        dismissWith('timer');
        delete globalState$$1.timeout;
      }, innerParams.timer);

      if (innerParams.timerProgressBar) {
        show(timerProgressBar);
        setTimeout(() => {
          if (globalState$$1.timeout && globalState$$1.timeout.running) {
            // timer can be already stopped or unset at this point
            animateTimerProgressBar(innerParams.timer);
          }
        });
      }
    }
  };

  const initFocus = (domCache, innerParams) => {
    if (innerParams.toast) {
      return;
    }

    if (!callIfFunction(innerParams.allowEnterKey)) {
      return blurActiveElement();
    }

    if (!focusButton(domCache, innerParams)) {
      setFocus(innerParams, -1, 1);
    }
  };

  const focusButton = (domCache, innerParams) => {
    if (innerParams.focusDeny && isVisible(domCache.denyButton)) {
      domCache.denyButton.focus();
      return true;
    }

    if (innerParams.focusCancel && isVisible(domCache.cancelButton)) {
      domCache.cancelButton.focus();
      return true;
    }

    if (innerParams.focusConfirm && isVisible(domCache.confirmButton)) {
      domCache.confirmButton.focus();
      return true;
    }

    return false;
  };

  const blurActiveElement = () => {
    if (document.activeElement && typeof document.activeElement.blur === 'function') {
      document.activeElement.blur();
    }
  };

  /**
   * Updates popup parameters.
   */

  function update(params) {
    const popup = getPopup();
    const innerParams = privateProps.innerParams.get(this);

    if (!popup || hasClass(popup, innerParams.hideClass.popup)) {
      return warn("You're trying to update the closed or closing popup, that won't work. Use the update() method in preConfirm parameter or show a new popup.");
    }

    const validUpdatableParams = {}; // assign valid params from `params` to `defaults`

    Object.keys(params).forEach(param => {
      if (Swal.isUpdatableParameter(param)) {
        validUpdatableParams[param] = params[param];
      } else {
        warn("Invalid parameter to update: \"".concat(param, "\". Updatable params are listed here: https://github.com/sweetalert2/sweetalert2/blob/master/src/utils/params.js\n\nIf you think this parameter should be updatable, request it here: https://github.com/sweetalert2/sweetalert2/issues/new?template=02_feature_request.md"));
      }
    });
    const updatedParams = Object.assign({}, innerParams, validUpdatableParams);
    render(this, updatedParams);
    privateProps.innerParams.set(this, updatedParams);
    Object.defineProperties(this, {
      params: {
        value: Object.assign({}, this.params, params),
        writable: false,
        enumerable: true
      }
    });
  }

  function _destroy() {
    const domCache = privateProps.domCache.get(this);
    const innerParams = privateProps.innerParams.get(this);

    if (!innerParams) {
      disposeWeakMaps(this); // The WeakMaps might have been partly destroyed, we must recall it to dispose any remaining weakmaps #2335

      return; // This instance has already been destroyed
    } // Check if there is another Swal closing


    if (domCache.popup && globalState.swalCloseEventFinishedCallback) {
      globalState.swalCloseEventFinishedCallback();
      delete globalState.swalCloseEventFinishedCallback;
    } // Check if there is a swal disposal defer timer


    if (globalState.deferDisposalTimer) {
      clearTimeout(globalState.deferDisposalTimer);
      delete globalState.deferDisposalTimer;
    }

    if (typeof innerParams.didDestroy === 'function') {
      innerParams.didDestroy();
    }

    disposeSwal(this);
  }

  const disposeSwal = instance => {
    disposeWeakMaps(instance); // Unset this.params so GC will dispose it (#1569)

    delete instance.params; // Unset globalState props so GC will dispose globalState (#1569)

    delete globalState.keydownHandler;
    delete globalState.keydownTarget; // Unset currentInstance

    delete globalState.currentInstance;
  };

  const disposeWeakMaps = instance => {
    // If the current instance is awaiting a promise result, we keep the privateMethods to call them once the promise result is retrieved #2335
    if (instance.isAwaitingPromise()) {
      unsetWeakMaps(privateProps, instance);
      privateProps.awaitingPromise.set(instance, true);
    } else {
      unsetWeakMaps(privateMethods, instance);
      unsetWeakMaps(privateProps, instance);
    }
  };

  const unsetWeakMaps = (obj, instance) => {
    for (const i in obj) {
      obj[i].delete(instance);
    }
  };



  var instanceMethods = /*#__PURE__*/Object.freeze({
    hideLoading: hideLoading,
    disableLoading: hideLoading,
    getInput: getInput$1,
    close: close,
    isAwaitingPromise: isAwaitingPromise,
    rejectPromise: rejectPromise,
    closePopup: close,
    closeModal: close,
    closeToast: close,
    enableButtons: enableButtons,
    disableButtons: disableButtons,
    enableInput: enableInput,
    disableInput: disableInput,
    showValidationMessage: showValidationMessage,
    resetValidationMessage: resetValidationMessage$1,
    getProgressSteps: getProgressSteps$1,
    _main: _main,
    update: update,
    _destroy: _destroy
  });

  let currentInstance;

  class SweetAlert {
    constructor() {
      // Prevent run in Node env
      if (typeof window === 'undefined') {
        return;
      }

      currentInstance = this;

      for (var _len = arguments.length, args = new Array(_len), _key = 0; _key < _len; _key++) {
        args[_key] = arguments[_key];
      }

      const outerParams = Object.freeze(this.constructor.argsToParams(args));
      Object.defineProperties(this, {
        params: {
          value: outerParams,
          writable: false,
          enumerable: true,
          configurable: true
        }
      });

      const promise = this._main(this.params);

      privateProps.promise.set(this, promise);
    } // `catch` cannot be the name of a module export, so we define our thenable methods here instead


    then(onFulfilled) {
      const promise = privateProps.promise.get(this);
      return promise.then(onFulfilled);
    }

    finally(onFinally) {
      const promise = privateProps.promise.get(this);
      return promise.finally(onFinally);
    }

  } // Assign instance methods from src/instanceMethods/*.js to prototype


  Object.assign(SweetAlert.prototype, instanceMethods); // Assign static methods from src/staticMethods/*.js to constructor

  Object.assign(SweetAlert, staticMethods); // Proxy to instance methods to constructor, for now, for backwards compatibility

  Object.keys(instanceMethods).forEach(key => {
    SweetAlert[key] = function () {
      if (currentInstance) {
        return currentInstance[key](...arguments);
      }
    };
  });
  SweetAlert.DismissReason = DismissReason;
  SweetAlert.version = '11.3.0';

  const Swal = SweetAlert;
  Swal.default = Swal;

  return Swal;

}));
if (typeof this !== 'undefined' && this.Sweetalert2){  this.swal = this.sweetAlert = this.Swal = this.SweetAlert = this.Sweetalert2}

"undefined"!=typeof document&&function(e,t){var n=e.createElement("style");if(e.getElementsByTagName("head")[0].appendChild(n),n.styleSheet)n.styleSheet.disabled||(n.styleSheet.cssText=t);else try{n.innerHTML=t}catch(e){n.innerText=t}}(document,".swal2-popup.swal2-toast{box-sizing:border-box;grid-column:1/4!important;grid-row:1/4!important;grid-template-columns:1fr 99fr 1fr;padding:1em;overflow-y:hidden;background:#fff;box-shadow:0 0 1px rgba(0,0,0,.075),0 1px 2px rgba(0,0,0,.075),1px 2px 4px rgba(0,0,0,.075),1px 3px 8px rgba(0,0,0,.075),2px 4px 16px rgba(0,0,0,.075);pointer-events:all}.swal2-popup.swal2-toast>*{grid-column:2}.swal2-popup.swal2-toast .swal2-title{margin:.5em 1em;padding:0;font-size:1em;text-align:initial}.swal2-popup.swal2-toast .swal2-loading{justify-content:center}.swal2-popup.swal2-toast .swal2-input{height:2em;margin:.5em;font-size:1em}.swal2-popup.swal2-toast .swal2-validation-message{font-size:1em}.swal2-popup.swal2-toast .swal2-footer{margin:.5em 0 0;padding:.5em 0 0;font-size:.8em}.swal2-popup.swal2-toast .swal2-close{grid-column:3/3;grid-row:1/99;align-self:center;width:.8em;height:.8em;margin:0;font-size:2em}.swal2-popup.swal2-toast .swal2-html-container{margin:.5em 1em;padding:0;font-size:1em;text-align:initial}.swal2-popup.swal2-toast .swal2-html-container:empty{padding:0}.swal2-popup.swal2-toast .swal2-loader{grid-column:1;grid-row:1/99;align-self:center;width:2em;height:2em;margin:.25em}.swal2-popup.swal2-toast .swal2-icon{grid-column:1;grid-row:1/99;align-self:center;width:2em;min-width:2em;height:2em;margin:0 .5em 0 0}.swal2-popup.swal2-toast .swal2-icon .swal2-icon-content{display:flex;align-items:center;font-size:1.8em;font-weight:700}.swal2-popup.swal2-toast .swal2-icon.swal2-success .swal2-success-ring{width:2em;height:2em}.swal2-popup.swal2-toast .swal2-icon.swal2-error [class^=swal2-x-mark-line]{top:.875em;width:1.375em}.swal2-popup.swal2-toast .swal2-icon.swal2-error [class^=swal2-x-mark-line][class$=left]{left:.3125em}.swal2-popup.swal2-toast .swal2-icon.swal2-error [class^=swal2-x-mark-line][class$=right]{right:.3125em}.swal2-popup.swal2-toast .swal2-actions{justify-content:flex-start;height:auto;margin:0;margin-top:.5em;padding:0 .5em}.swal2-popup.swal2-toast .swal2-styled{margin:.25em .5em;padding:.4em .6em;font-size:1em}.swal2-popup.swal2-toast .swal2-success{border-color:#a5dc86}.swal2-popup.swal2-toast .swal2-success [class^=swal2-success-circular-line]{position:absolute;width:1.6em;height:3em;transform:rotate(45deg);border-radius:50%}.swal2-popup.swal2-toast .swal2-success [class^=swal2-success-circular-line][class$=left]{top:-.8em;left:-.5em;transform:rotate(-45deg);transform-origin:2em 2em;border-radius:4em 0 0 4em}.swal2-popup.swal2-toast .swal2-success [class^=swal2-success-circular-line][class$=right]{top:-.25em;left:.9375em;transform-origin:0 1.5em;border-radius:0 4em 4em 0}.swal2-popup.swal2-toast .swal2-success .swal2-success-ring{width:2em;height:2em}.swal2-popup.swal2-toast .swal2-success .swal2-success-fix{top:0;left:.4375em;width:.4375em;height:2.6875em}.swal2-popup.swal2-toast .swal2-success [class^=swal2-success-line]{height:.3125em}.swal2-popup.swal2-toast .swal2-success [class^=swal2-success-line][class$=tip]{top:1.125em;left:.1875em;width:.75em}.swal2-popup.swal2-toast .swal2-success [class^=swal2-success-line][class$=long]{top:.9375em;right:.1875em;width:1.375em}.swal2-popup.swal2-toast .swal2-success.swal2-icon-show .swal2-success-line-tip{-webkit-animation:swal2-toast-animate-success-line-tip .75s;animation:swal2-toast-animate-success-line-tip .75s}.swal2-popup.swal2-toast .swal2-success.swal2-icon-show .swal2-success-line-long{-webkit-animation:swal2-toast-animate-success-line-long .75s;animation:swal2-toast-animate-success-line-long .75s}.swal2-popup.swal2-toast.swal2-show{-webkit-animation:swal2-toast-show .5s;animation:swal2-toast-show .5s}.swal2-popup.swal2-toast.swal2-hide{-webkit-animation:swal2-toast-hide .1s forwards;animation:swal2-toast-hide .1s forwards}.swal2-container{display:grid;position:fixed;z-index:1060;top:0;right:0;bottom:0;left:0;box-sizing:border-box;grid-template-areas:\"top-start     top            top-end\" \"center-start  center         center-end\" \"bottom-start  bottom-center  bottom-end\";grid-template-rows:minmax(-webkit-min-content,auto) minmax(-webkit-min-content,auto) minmax(-webkit-min-content,auto);grid-template-rows:minmax(min-content,auto) minmax(min-content,auto) minmax(min-content,auto);height:100%;padding:.625em;overflow-x:hidden;transition:background-color .1s;-webkit-overflow-scrolling:touch}.swal2-container.swal2-backdrop-show,.swal2-container.swal2-noanimation{background:rgba(0,0,0,.4)}.swal2-container.swal2-backdrop-hide{background:0 0!important}.swal2-container.swal2-bottom-start,.swal2-container.swal2-center-start,.swal2-container.swal2-top-start{grid-template-columns:minmax(0,1fr) auto auto}.swal2-container.swal2-bottom,.swal2-container.swal2-center,.swal2-container.swal2-top{grid-template-columns:auto minmax(0,1fr) auto}.swal2-container.swal2-bottom-end,.swal2-container.swal2-center-end,.swal2-container.swal2-top-end{grid-template-columns:auto auto minmax(0,1fr)}.swal2-container.swal2-top-start>.swal2-popup{align-self:start}.swal2-container.swal2-top>.swal2-popup{grid-column:2;align-self:start;justify-self:center}.swal2-container.swal2-top-end>.swal2-popup,.swal2-container.swal2-top-right>.swal2-popup{grid-column:3;align-self:start;justify-self:end}.swal2-container.swal2-center-left>.swal2-popup,.swal2-container.swal2-center-start>.swal2-popup{grid-row:2;align-self:center}.swal2-container.swal2-center>.swal2-popup{grid-column:2;grid-row:2;align-self:center;justify-self:center}.swal2-container.swal2-center-end>.swal2-popup,.swal2-container.swal2-center-right>.swal2-popup{grid-column:3;grid-row:2;align-self:center;justify-self:end}.swal2-container.swal2-bottom-left>.swal2-popup,.swal2-container.swal2-bottom-start>.swal2-popup{grid-column:1;grid-row:3;align-self:end}.swal2-container.swal2-bottom>.swal2-popup{grid-column:2;grid-row:3;justify-self:center;align-self:end}.swal2-container.swal2-bottom-end>.swal2-popup,.swal2-container.swal2-bottom-right>.swal2-popup{grid-column:3;grid-row:3;align-self:end;justify-self:end}.swal2-container.swal2-grow-fullscreen>.swal2-popup,.swal2-container.swal2-grow-row>.swal2-popup{grid-column:1/4;width:100%}.swal2-container.swal2-grow-column>.swal2-popup,.swal2-container.swal2-grow-fullscreen>.swal2-popup{grid-row:1/4;align-self:stretch}.swal2-container.swal2-no-transition{transition:none!important}.swal2-popup{display:none;position:relative;box-sizing:border-box;grid-template-columns:minmax(0,100%);width:32em;max-width:100%;padding:0 0 1.25em;border:none;border-radius:5px;background:#fff;color:#545454;font-family:inherit;font-size:1rem}.swal2-popup:focus{outline:0}.swal2-popup.swal2-loading{overflow-y:hidden}.swal2-title{position:relative;max-width:100%;margin:0;padding:.8em 1em 0;color:inherit;font-size:1.875em;font-weight:600;text-align:center;text-transform:none;word-wrap:break-word}.swal2-actions{display:flex;z-index:1;box-sizing:border-box;flex-wrap:wrap;align-items:center;justify-content:center;width:auto;margin:1.25em auto 0;padding:0}.swal2-actions:not(.swal2-loading) .swal2-styled[disabled]{opacity:.4}.swal2-actions:not(.swal2-loading) .swal2-styled:hover{background-image:linear-gradient(rgba(0,0,0,.1),rgba(0,0,0,.1))}.swal2-actions:not(.swal2-loading) .swal2-styled:active{background-image:linear-gradient(rgba(0,0,0,.2),rgba(0,0,0,.2))}.swal2-loader{display:none;align-items:center;justify-content:center;width:2.2em;height:2.2em;margin:0 1.875em;-webkit-animation:swal2-rotate-loading 1.5s linear 0s infinite normal;animation:swal2-rotate-loading 1.5s linear 0s infinite normal;border-width:.25em;border-style:solid;border-radius:100%;border-color:#2778c4 transparent #2778c4 transparent}.swal2-styled{margin:.3125em;padding:.625em 1.1em;transition:box-shadow .1s;box-shadow:0 0 0 3px transparent;font-weight:500}.swal2-styled:not([disabled]){cursor:pointer}.swal2-styled.swal2-confirm{border:0;border-radius:.25em;background:initial;background-color:#7066e0;color:#fff;font-size:1em}.swal2-styled.swal2-confirm:focus{box-shadow:0 0 0 3px rgba(112,102,224,.5)}.swal2-styled.swal2-deny{border:0;border-radius:.25em;background:initial;background-color:#dc3741;color:#fff;font-size:1em}.swal2-styled.swal2-deny:focus{box-shadow:0 0 0 3px rgba(220,55,65,.5)}.swal2-styled.swal2-cancel{border:0;border-radius:.25em;background:initial;background-color:#6e7881;color:#fff;font-size:1em}.swal2-styled.swal2-cancel:focus{box-shadow:0 0 0 3px rgba(110,120,129,.5)}.swal2-styled.swal2-default-outline:focus{box-shadow:0 0 0 3px rgba(100,150,200,.5)}.swal2-styled:focus{outline:0}.swal2-styled::-moz-focus-inner{border:0}.swal2-footer{justify-content:center;margin:1em 0 0;padding:1em 1em 0;border-top:1px solid #eee;color:inherit;font-size:1em}.swal2-timer-progress-bar-container{position:absolute;right:0;bottom:0;left:0;grid-column:auto!important;height:.25em;overflow:hidden;border-bottom-right-radius:5px;border-bottom-left-radius:5px}.swal2-timer-progress-bar{width:100%;height:.25em;background:rgba(0,0,0,.2)}.swal2-image{max-width:100%;margin:2em auto 1em}.swal2-close{z-index:2;align-items:center;justify-content:center;width:1.2em;height:1.2em;margin-top:0;margin-right:0;margin-bottom:-1.2em;padding:0;overflow:hidden;transition:color .1s,box-shadow .1s;border:none;border-radius:5px;background:0 0;color:#ccc;font-family:serif;font-family:monospace;font-size:2.5em;cursor:pointer;justify-self:end}.swal2-close:hover{transform:none;background:0 0;color:#f27474}.swal2-close:focus{outline:0;box-shadow:inset 0 0 0 3px rgba(100,150,200,.5)}.swal2-close::-moz-focus-inner{border:0}.swal2-html-container{z-index:1;justify-content:center;margin:1em 1.6em .3em;padding:0;overflow:auto;color:inherit;font-size:1.125em;font-weight:400;line-height:normal;text-align:center;word-wrap:break-word;word-break:break-word}.swal2-checkbox,.swal2-file,.swal2-input,.swal2-radio,.swal2-select,.swal2-textarea{margin:1em 2em 0}.swal2-file,.swal2-input,.swal2-textarea{box-sizing:border-box;width:auto;transition:border-color .1s,box-shadow .1s;border:1px solid #d9d9d9;border-radius:.1875em;background:inherit;box-shadow:inset 0 1px 1px rgba(0,0,0,.06),0 0 0 3px transparent;color:inherit;font-size:1.125em}.swal2-file.swal2-inputerror,.swal2-input.swal2-inputerror,.swal2-textarea.swal2-inputerror{border-color:#f27474!important;box-shadow:0 0 2px #f27474!important}.swal2-file:focus,.swal2-input:focus,.swal2-textarea:focus{border:1px solid #b4dbed;outline:0;box-shadow:inset 0 1px 1px rgba(0,0,0,.06),0 0 0 3px rgba(100,150,200,.5)}.swal2-file::-moz-placeholder,.swal2-input::-moz-placeholder,.swal2-textarea::-moz-placeholder{color:#ccc}.swal2-file:-ms-input-placeholder,.swal2-input:-ms-input-placeholder,.swal2-textarea:-ms-input-placeholder{color:#ccc}.swal2-file::placeholder,.swal2-input::placeholder,.swal2-textarea::placeholder{color:#ccc}.swal2-range{margin:1em 2em 0;background:#fff}.swal2-range input{width:80%}.swal2-range output{width:20%;color:inherit;font-weight:600;text-align:center}.swal2-range input,.swal2-range output{height:2.625em;padding:0;font-size:1.125em;line-height:2.625em}.swal2-input{height:2.625em;padding:0 .75em}.swal2-file{width:75%;margin-right:auto;margin-left:auto;background:inherit;font-size:1.125em}.swal2-textarea{height:6.75em;padding:.75em}.swal2-select{min-width:50%;max-width:100%;padding:.375em .625em;background:inherit;color:inherit;font-size:1.125em}.swal2-checkbox,.swal2-radio{align-items:center;justify-content:center;background:#fff;color:inherit}.swal2-checkbox label,.swal2-radio label{margin:0 .6em;font-size:1.125em}.swal2-checkbox input,.swal2-radio input{flex-shrink:0;margin:0 .4em}.swal2-input-label{display:flex;justify-content:center;margin:1em auto 0}.swal2-validation-message{align-items:center;justify-content:center;margin:1em 0 0;padding:.625em;overflow:hidden;background:#f0f0f0;color:#666;font-size:1em;font-weight:300}.swal2-validation-message::before{content:\"!\";display:inline-block;width:1.5em;min-width:1.5em;height:1.5em;margin:0 .625em;border-radius:50%;background-color:#f27474;color:#fff;font-weight:600;line-height:1.5em;text-align:center}.swal2-icon{position:relative;box-sizing:content-box;justify-content:center;width:5em;height:5em;margin:2.5em auto .6em;border:.25em solid transparent;border-radius:50%;border-color:#000;font-family:inherit;line-height:5em;cursor:default;-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none}.swal2-icon .swal2-icon-content{display:flex;align-items:center;font-size:3.75em}.swal2-icon.swal2-error{border-color:#f27474;color:#f27474}.swal2-icon.swal2-error .swal2-x-mark{position:relative;flex-grow:1}.swal2-icon.swal2-error [class^=swal2-x-mark-line]{display:block;position:absolute;top:2.3125em;width:2.9375em;height:.3125em;border-radius:.125em;background-color:#f27474}.swal2-icon.swal2-error [class^=swal2-x-mark-line][class$=left]{left:1.0625em;transform:rotate(45deg)}.swal2-icon.swal2-error [class^=swal2-x-mark-line][class$=right]{right:1em;transform:rotate(-45deg)}.swal2-icon.swal2-error.swal2-icon-show{-webkit-animation:swal2-animate-error-icon .5s;animation:swal2-animate-error-icon .5s}.swal2-icon.swal2-error.swal2-icon-show .swal2-x-mark{-webkit-animation:swal2-animate-error-x-mark .5s;animation:swal2-animate-error-x-mark .5s}.swal2-icon.swal2-warning{border-color:#facea8;color:#f8bb86}.swal2-icon.swal2-warning.swal2-icon-show{-webkit-animation:swal2-animate-error-icon .5s;animation:swal2-animate-error-icon .5s}.swal2-icon.swal2-warning.swal2-icon-show .swal2-icon-content{-webkit-animation:swal2-animate-i-mark .5s;animation:swal2-animate-i-mark .5s}.swal2-icon.swal2-info{border-color:#9de0f6;color:#3fc3ee}.swal2-icon.swal2-info.swal2-icon-show{-webkit-animation:swal2-animate-error-icon .5s;animation:swal2-animate-error-icon .5s}.swal2-icon.swal2-info.swal2-icon-show .swal2-icon-content{-webkit-animation:swal2-animate-i-mark .8s;animation:swal2-animate-i-mark .8s}.swal2-icon.swal2-question{border-color:#c9dae1;color:#87adbd}.swal2-icon.swal2-question.swal2-icon-show{-webkit-animation:swal2-animate-error-icon .5s;animation:swal2-animate-error-icon .5s}.swal2-icon.swal2-question.swal2-icon-show .swal2-icon-content{-webkit-animation:swal2-animate-question-mark .8s;animation:swal2-animate-question-mark .8s}.swal2-icon.swal2-success{border-color:#a5dc86;color:#a5dc86}.swal2-icon.swal2-success [class^=swal2-success-circular-line]{position:absolute;width:3.75em;height:7.5em;transform:rotate(45deg);border-radius:50%}.swal2-icon.swal2-success [class^=swal2-success-circular-line][class$=left]{top:-.4375em;left:-2.0635em;transform:rotate(-45deg);transform-origin:3.75em 3.75em;border-radius:7.5em 0 0 7.5em}.swal2-icon.swal2-success [class^=swal2-success-circular-line][class$=right]{top:-.6875em;left:1.875em;transform:rotate(-45deg);transform-origin:0 3.75em;border-radius:0 7.5em 7.5em 0}.swal2-icon.swal2-success .swal2-success-ring{position:absolute;z-index:2;top:-.25em;left:-.25em;box-sizing:content-box;width:100%;height:100%;border:.25em solid rgba(165,220,134,.3);border-radius:50%}.swal2-icon.swal2-success .swal2-success-fix{position:absolute;z-index:1;top:.5em;left:1.625em;width:.4375em;height:5.625em;transform:rotate(-45deg)}.swal2-icon.swal2-success [class^=swal2-success-line]{display:block;position:absolute;z-index:2;height:.3125em;border-radius:.125em;background-color:#a5dc86}.swal2-icon.swal2-success [class^=swal2-success-line][class$=tip]{top:2.875em;left:.8125em;width:1.5625em;transform:rotate(45deg)}.swal2-icon.swal2-success [class^=swal2-success-line][class$=long]{top:2.375em;right:.5em;width:2.9375em;transform:rotate(-45deg)}.swal2-icon.swal2-success.swal2-icon-show .swal2-success-line-tip{-webkit-animation:swal2-animate-success-line-tip .75s;animation:swal2-animate-success-line-tip .75s}.swal2-icon.swal2-success.swal2-icon-show .swal2-success-line-long{-webkit-animation:swal2-animate-success-line-long .75s;animation:swal2-animate-success-line-long .75s}.swal2-icon.swal2-success.swal2-icon-show .swal2-success-circular-line-right{-webkit-animation:swal2-rotate-success-circular-line 4.25s ease-in;animation:swal2-rotate-success-circular-line 4.25s ease-in}.swal2-progress-steps{flex-wrap:wrap;align-items:center;max-width:100%;margin:1.25em auto;padding:0;background:inherit;font-weight:600}.swal2-progress-steps li{display:inline-block;position:relative}.swal2-progress-steps .swal2-progress-step{z-index:20;flex-shrink:0;width:2em;height:2em;border-radius:2em;background:#2778c4;color:#fff;line-height:2em;text-align:center}.swal2-progress-steps .swal2-progress-step.swal2-active-progress-step{background:#2778c4}.swal2-progress-steps .swal2-progress-step.swal2-active-progress-step~.swal2-progress-step{background:#add8e6;color:#fff}.swal2-progress-steps .swal2-progress-step.swal2-active-progress-step~.swal2-progress-step-line{background:#add8e6}.swal2-progress-steps .swal2-progress-step-line{z-index:10;flex-shrink:0;width:2.5em;height:.4em;margin:0 -1px;background:#2778c4}[class^=swal2]{-webkit-tap-highlight-color:transparent}.swal2-show{-webkit-animation:swal2-show .3s;animation:swal2-show .3s}.swal2-hide{-webkit-animation:swal2-hide .15s forwards;animation:swal2-hide .15s forwards}.swal2-noanimation{transition:none}.swal2-scrollbar-measure{position:absolute;top:-9999px;width:50px;height:50px;overflow:scroll}.swal2-rtl .swal2-close{margin-right:initial;margin-left:0}.swal2-rtl .swal2-timer-progress-bar{right:0;left:auto}@-webkit-keyframes swal2-toast-show{0%{transform:translateY(-.625em) rotateZ(2deg)}33%{transform:translateY(0) rotateZ(-2deg)}66%{transform:translateY(.3125em) rotateZ(2deg)}100%{transform:translateY(0) rotateZ(0)}}@keyframes swal2-toast-show{0%{transform:translateY(-.625em) rotateZ(2deg)}33%{transform:translateY(0) rotateZ(-2deg)}66%{transform:translateY(.3125em) rotateZ(2deg)}100%{transform:translateY(0) rotateZ(0)}}@-webkit-keyframes swal2-toast-hide{100%{transform:rotateZ(1deg);opacity:0}}@keyframes swal2-toast-hide{100%{transform:rotateZ(1deg);opacity:0}}@-webkit-keyframes swal2-toast-animate-success-line-tip{0%{top:.5625em;left:.0625em;width:0}54%{top:.125em;left:.125em;width:0}70%{top:.625em;left:-.25em;width:1.625em}84%{top:1.0625em;left:.75em;width:.5em}100%{top:1.125em;left:.1875em;width:.75em}}@keyframes swal2-toast-animate-success-line-tip{0%{top:.5625em;left:.0625em;width:0}54%{top:.125em;left:.125em;width:0}70%{top:.625em;left:-.25em;width:1.625em}84%{top:1.0625em;left:.75em;width:.5em}100%{top:1.125em;left:.1875em;width:.75em}}@-webkit-keyframes swal2-toast-animate-success-line-long{0%{top:1.625em;right:1.375em;width:0}65%{top:1.25em;right:.9375em;width:0}84%{top:.9375em;right:0;width:1.125em}100%{top:.9375em;right:.1875em;width:1.375em}}@keyframes swal2-toast-animate-success-line-long{0%{top:1.625em;right:1.375em;width:0}65%{top:1.25em;right:.9375em;width:0}84%{top:.9375em;right:0;width:1.125em}100%{top:.9375em;right:.1875em;width:1.375em}}@-webkit-keyframes swal2-show{0%{transform:scale(.7)}45%{transform:scale(1.05)}80%{transform:scale(.95)}100%{transform:scale(1)}}@keyframes swal2-show{0%{transform:scale(.7)}45%{transform:scale(1.05)}80%{transform:scale(.95)}100%{transform:scale(1)}}@-webkit-keyframes swal2-hide{0%{transform:scale(1);opacity:1}100%{transform:scale(.5);opacity:0}}@keyframes swal2-hide{0%{transform:scale(1);opacity:1}100%{transform:scale(.5);opacity:0}}@-webkit-keyframes swal2-animate-success-line-tip{0%{top:1.1875em;left:.0625em;width:0}54%{top:1.0625em;left:.125em;width:0}70%{top:2.1875em;left:-.375em;width:3.125em}84%{top:3em;left:1.3125em;width:1.0625em}100%{top:2.8125em;left:.8125em;width:1.5625em}}@keyframes swal2-animate-success-line-tip{0%{top:1.1875em;left:.0625em;width:0}54%{top:1.0625em;left:.125em;width:0}70%{top:2.1875em;left:-.375em;width:3.125em}84%{top:3em;left:1.3125em;width:1.0625em}100%{top:2.8125em;left:.8125em;width:1.5625em}}@-webkit-keyframes swal2-animate-success-line-long{0%{top:3.375em;right:2.875em;width:0}65%{top:3.375em;right:2.875em;width:0}84%{top:2.1875em;right:0;width:3.4375em}100%{top:2.375em;right:.5em;width:2.9375em}}@keyframes swal2-animate-success-line-long{0%{top:3.375em;right:2.875em;width:0}65%{top:3.375em;right:2.875em;width:0}84%{top:2.1875em;right:0;width:3.4375em}100%{top:2.375em;right:.5em;width:2.9375em}}@-webkit-keyframes swal2-rotate-success-circular-line{0%{transform:rotate(-45deg)}5%{transform:rotate(-45deg)}12%{transform:rotate(-405deg)}100%{transform:rotate(-405deg)}}@keyframes swal2-rotate-success-circular-line{0%{transform:rotate(-45deg)}5%{transform:rotate(-45deg)}12%{transform:rotate(-405deg)}100%{transform:rotate(-405deg)}}@-webkit-keyframes swal2-animate-error-x-mark{0%{margin-top:1.625em;transform:scale(.4);opacity:0}50%{margin-top:1.625em;transform:scale(.4);opacity:0}80%{margin-top:-.375em;transform:scale(1.15)}100%{margin-top:0;transform:scale(1);opacity:1}}@keyframes swal2-animate-error-x-mark{0%{margin-top:1.625em;transform:scale(.4);opacity:0}50%{margin-top:1.625em;transform:scale(.4);opacity:0}80%{margin-top:-.375em;transform:scale(1.15)}100%{margin-top:0;transform:scale(1);opacity:1}}@-webkit-keyframes swal2-animate-error-icon{0%{transform:rotateX(100deg);opacity:0}100%{transform:rotateX(0);opacity:1}}@keyframes swal2-animate-error-icon{0%{transform:rotateX(100deg);opacity:0}100%{transform:rotateX(0);opacity:1}}@-webkit-keyframes swal2-rotate-loading{0%{transform:rotate(0)}100%{transform:rotate(360deg)}}@keyframes swal2-rotate-loading{0%{transform:rotate(0)}100%{transform:rotate(360deg)}}@-webkit-keyframes swal2-animate-question-mark{0%{transform:rotateY(-360deg)}100%{transform:rotateY(0)}}@keyframes swal2-animate-question-mark{0%{transform:rotateY(-360deg)}100%{transform:rotateY(0)}}@-webkit-keyframes swal2-animate-i-mark{0%{transform:rotateZ(45deg);opacity:0}25%{transform:rotateZ(-25deg);opacity:.4}50%{transform:rotateZ(15deg);opacity:.8}75%{transform:rotateZ(-5deg);opacity:1}100%{transform:rotateX(0);opacity:1}}@keyframes swal2-animate-i-mark{0%{transform:rotateZ(45deg);opacity:0}25%{transform:rotateZ(-25deg);opacity:.4}50%{transform:rotateZ(15deg);opacity:.8}75%{transform:rotateZ(-5deg);opacity:1}100%{transform:rotateX(0);opacity:1}}body.swal2-shown:not(.swal2-no-backdrop):not(.swal2-toast-shown){overflow:hidden}body.swal2-height-auto{height:auto!important}body.swal2-no-backdrop .swal2-container{background-color:transparent!important;pointer-events:none}body.swal2-no-backdrop .swal2-container .swal2-popup{pointer-events:all}body.swal2-no-backdrop .swal2-container .swal2-modal{box-shadow:0 0 10px rgba(0,0,0,.4)}@media print{body.swal2-shown:not(.swal2-no-backdrop):not(.swal2-toast-shown){overflow-y:scroll!important}body.swal2-shown:not(.swal2-no-backdrop):not(.swal2-toast-shown)>[aria-hidden=true]{display:none}body.swal2-shown:not(.swal2-no-backdrop):not(.swal2-toast-shown) .swal2-container{position:static!important}}body.swal2-toast-shown .swal2-container{box-sizing:border-box;width:360px;max-width:100%;background-color:transparent;pointer-events:none}body.swal2-toast-shown .swal2-container.swal2-top{top:0;right:auto;bottom:auto;left:50%;transform:translateX(-50%)}body.swal2-toast-shown .swal2-container.swal2-top-end,body.swal2-toast-shown .swal2-container.swal2-top-right{top:0;right:0;bottom:auto;left:auto}body.swal2-toast-shown .swal2-container.swal2-top-left,body.swal2-toast-shown .swal2-container.swal2-top-start{top:0;right:auto;bottom:auto;left:0}body.swal2-toast-shown .swal2-container.swal2-center-left,body.swal2-toast-shown .swal2-container.swal2-center-start{top:50%;right:auto;bottom:auto;left:0;transform:translateY(-50%)}body.swal2-toast-shown .swal2-container.swal2-center{top:50%;right:auto;bottom:auto;left:50%;transform:translate(-50%,-50%)}body.swal2-toast-shown .swal2-container.swal2-center-end,body.swal2-toast-shown .swal2-container.swal2-center-right{top:50%;right:0;bottom:auto;left:auto;transform:translateY(-50%)}body.swal2-toast-shown .swal2-container.swal2-bottom-left,body.swal2-toast-shown .swal2-container.swal2-bottom-start{top:auto;right:auto;bottom:0;left:0}body.swal2-toast-shown .swal2-container.swal2-bottom{top:auto;right:auto;bottom:0;left:50%;transform:translateX(-50%)}body.swal2-toast-shown .swal2-container.swal2-bottom-end,body.swal2-toast-shown .swal2-container.swal2-bottom-right{top:auto;right:0;bottom:0;left:auto}");

/***/ }),

/***/ "./node_modules/toggle-selection/index.js":
/*!************************************************!*\
  !*** ./node_modules/toggle-selection/index.js ***!
  \************************************************/
/***/ ((module) => {


module.exports = function () {
  var selection = document.getSelection();
  if (!selection.rangeCount) {
    return function () {};
  }
  var active = document.activeElement;

  var ranges = [];
  for (var i = 0; i < selection.rangeCount; i++) {
    ranges.push(selection.getRangeAt(i));
  }

  switch (active.tagName.toUpperCase()) { // .toUpperCase handles XHTML
    case 'INPUT':
    case 'TEXTAREA':
      active.blur();
      break;

    default:
      active = null;
      break;
  }

  selection.removeAllRanges();
  return function () {
    selection.type === 'Caret' &&
    selection.removeAllRanges();

    if (!selection.rangeCount) {
      ranges.forEach(function(range) {
        selection.addRange(range);
      });
    }

    active &&
    active.focus();
  };
};


/***/ }),

/***/ "./node_modules/vditor/dist/index.min.js":
/*!***********************************************!*\
  !*** ./node_modules/vditor/dist/index.min.js ***!
  \***********************************************/
/***/ (function(module) {

!function(e,t){ true?module.exports=t():0}(this,(function(){return(()=>{var e={694:e=>{var t=function(){this.Diff_Timeout=1,this.Diff_EditCost=4,this.Match_Threshold=.5,this.Match_Distance=1e3,this.Patch_DeleteThreshold=.5,this.Patch_Margin=4,this.Match_MaxBits=32},n=-1;t.Diff=function(e,t){return[e,t]},t.prototype.diff_main=function(e,n,r,i){void 0===i&&(i=this.Diff_Timeout<=0?Number.MAX_VALUE:(new Date).getTime()+1e3*this.Diff_Timeout);var o=i;if(null==e||null==n)throw new Error("Null input. (diff_main)");if(e==n)return e?[new t.Diff(0,e)]:[];void 0===r&&(r=!0);var a=r,l=this.diff_commonPrefix(e,n),s=e.substring(0,l);e=e.substring(l),n=n.substring(l),l=this.diff_commonSuffix(e,n);var d=e.substring(e.length-l);e=e.substring(0,e.length-l),n=n.substring(0,n.length-l);var c=this.diff_compute_(e,n,a,o);return s&&c.unshift(new t.Diff(0,s)),d&&c.push(new t.Diff(0,d)),this.diff_cleanupMerge(c),c},t.prototype.diff_compute_=function(e,r,i,o){var a;if(!e)return[new t.Diff(1,r)];if(!r)return[new t.Diff(n,e)];var l=e.length>r.length?e:r,s=e.length>r.length?r:e,d=l.indexOf(s);if(-1!=d)return a=[new t.Diff(1,l.substring(0,d)),new t.Diff(0,s),new t.Diff(1,l.substring(d+s.length))],e.length>r.length&&(a[0][0]=a[2][0]=n),a;if(1==s.length)return[new t.Diff(n,e),new t.Diff(1,r)];var c=this.diff_halfMatch_(e,r);if(c){var u=c[0],p=c[1],m=c[2],f=c[3],h=c[4],v=this.diff_main(u,m,i,o),g=this.diff_main(p,f,i,o);return v.concat([new t.Diff(0,h)],g)}return i&&e.length>100&&r.length>100?this.diff_lineMode_(e,r,o):this.diff_bisect_(e,r,o)},t.prototype.diff_lineMode_=function(e,r,i){var o=this.diff_linesToChars_(e,r);e=o.chars1,r=o.chars2;var a=o.lineArray,l=this.diff_main(e,r,!1,i);this.diff_charsToLines_(l,a),this.diff_cleanupSemantic(l),l.push(new t.Diff(0,""));for(var s=0,d=0,c=0,u="",p="";s<l.length;){switch(l[s][0]){case 1:c++,p+=l[s][1];break;case n:d++,u+=l[s][1];break;case 0:if(d>=1&&c>=1){l.splice(s-d-c,d+c),s=s-d-c;for(var m=this.diff_main(u,p,!1,i),f=m.length-1;f>=0;f--)l.splice(s,0,m[f]);s+=m.length}c=0,d=0,u="",p=""}s++}return l.pop(),l},t.prototype.diff_bisect_=function(e,r,i){for(var o=e.length,a=r.length,l=Math.ceil((o+a)/2),s=l,d=2*l,c=new Array(d),u=new Array(d),p=0;p<d;p++)c[p]=-1,u[p]=-1;c[s+1]=0,u[s+1]=0;for(var m=o-a,f=m%2!=0,h=0,v=0,g=0,y=0,b=0;b<l&&!((new Date).getTime()>i);b++){for(var w=-b+h;w<=b-v;w+=2){for(var E=s+w,k=(M=w==-b||w!=b&&c[E-1]<c[E+1]?c[E+1]:c[E-1]+1)-w;M<o&&k<a&&e.charAt(M)==r.charAt(k);)M++,k++;if(c[E]=M,M>o)v+=2;else if(k>a)h+=2;else if(f){if((L=s+m-w)>=0&&L<d&&-1!=u[L])if(M>=(C=o-u[L]))return this.diff_bisectSplit_(e,r,M,k,i)}}for(var S=-b+g;S<=b-y;S+=2){for(var C,L=s+S,T=(C=S==-b||S!=b&&u[L-1]<u[L+1]?u[L+1]:u[L-1]+1)-S;C<o&&T<a&&e.charAt(o-C-1)==r.charAt(a-T-1);)C++,T++;if(u[L]=C,C>o)y+=2;else if(T>a)g+=2;else if(!f){if((E=s+m-S)>=0&&E<d&&-1!=c[E]){var M;k=s+(M=c[E])-E;if(M>=(C=o-C))return this.diff_bisectSplit_(e,r,M,k,i)}}}}return[new t.Diff(n,e),new t.Diff(1,r)]},t.prototype.diff_bisectSplit_=function(e,t,n,r,i){var o=e.substring(0,n),a=t.substring(0,r),l=e.substring(n),s=t.substring(r),d=this.diff_main(o,a,!1,i),c=this.diff_main(l,s,!1,i);return d.concat(c)},t.prototype.diff_linesToChars_=function(e,t){var n=[],r={};function i(e){for(var t="",i=0,a=-1,l=n.length;a<e.length-1;){-1==(a=e.indexOf("\n",i))&&(a=e.length-1);var s=e.substring(i,a+1);(r.hasOwnProperty?r.hasOwnProperty(s):void 0!==r[s])?t+=String.fromCharCode(r[s]):(l==o&&(s=e.substring(i),a=e.length),t+=String.fromCharCode(l),r[s]=l,n[l++]=s),i=a+1}return t}n[0]="";var o=4e4,a=i(e);return o=65535,{chars1:a,chars2:i(t),lineArray:n}},t.prototype.diff_charsToLines_=function(e,t){for(var n=0;n<e.length;n++){for(var r=e[n][1],i=[],o=0;o<r.length;o++)i[o]=t[r.charCodeAt(o)];e[n][1]=i.join("")}},t.prototype.diff_commonPrefix=function(e,t){if(!e||!t||e.charAt(0)!=t.charAt(0))return 0;for(var n=0,r=Math.min(e.length,t.length),i=r,o=0;n<i;)e.substring(o,i)==t.substring(o,i)?o=n=i:r=i,i=Math.floor((r-n)/2+n);return i},t.prototype.diff_commonSuffix=function(e,t){if(!e||!t||e.charAt(e.length-1)!=t.charAt(t.length-1))return 0;for(var n=0,r=Math.min(e.length,t.length),i=r,o=0;n<i;)e.substring(e.length-i,e.length-o)==t.substring(t.length-i,t.length-o)?o=n=i:r=i,i=Math.floor((r-n)/2+n);return i},t.prototype.diff_commonOverlap_=function(e,t){var n=e.length,r=t.length;if(0==n||0==r)return 0;n>r?e=e.substring(n-r):n<r&&(t=t.substring(0,n));var i=Math.min(n,r);if(e==t)return i;for(var o=0,a=1;;){var l=e.substring(i-a),s=t.indexOf(l);if(-1==s)return o;a+=s,0!=s&&e.substring(i-a)!=t.substring(0,a)||(o=a,a++)}},t.prototype.diff_halfMatch_=function(e,t){if(this.Diff_Timeout<=0)return null;var n=e.length>t.length?e:t,r=e.length>t.length?t:e;if(n.length<4||2*r.length<n.length)return null;var i=this;function o(e,t,n){for(var r,o,a,l,s=e.substring(n,n+Math.floor(e.length/4)),d=-1,c="";-1!=(d=t.indexOf(s,d+1));){var u=i.diff_commonPrefix(e.substring(n),t.substring(d)),p=i.diff_commonSuffix(e.substring(0,n),t.substring(0,d));c.length<p+u&&(c=t.substring(d-p,d)+t.substring(d,d+u),r=e.substring(0,n-p),o=e.substring(n+u),a=t.substring(0,d-p),l=t.substring(d+u))}return 2*c.length>=e.length?[r,o,a,l,c]:null}var a,l,s,d,c,u=o(n,r,Math.ceil(n.length/4)),p=o(n,r,Math.ceil(n.length/2));return u||p?(a=p?u&&u[4].length>p[4].length?u:p:u,e.length>t.length?(l=a[0],s=a[1],d=a[2],c=a[3]):(d=a[0],c=a[1],l=a[2],s=a[3]),[l,s,d,c,a[4]]):null},t.prototype.diff_cleanupSemantic=function(e){for(var r=!1,i=[],o=0,a=null,l=0,s=0,d=0,c=0,u=0;l<e.length;)0==e[l][0]?(i[o++]=l,s=c,d=u,c=0,u=0,a=e[l][1]):(1==e[l][0]?c+=e[l][1].length:u+=e[l][1].length,a&&a.length<=Math.max(s,d)&&a.length<=Math.max(c,u)&&(e.splice(i[o-1],0,new t.Diff(n,a)),e[i[o-1]+1][0]=1,o--,l=--o>0?i[o-1]:-1,s=0,d=0,c=0,u=0,a=null,r=!0)),l++;for(r&&this.diff_cleanupMerge(e),this.diff_cleanupSemanticLossless(e),l=1;l<e.length;){if(e[l-1][0]==n&&1==e[l][0]){var p=e[l-1][1],m=e[l][1],f=this.diff_commonOverlap_(p,m),h=this.diff_commonOverlap_(m,p);f>=h?(f>=p.length/2||f>=m.length/2)&&(e.splice(l,0,new t.Diff(0,m.substring(0,f))),e[l-1][1]=p.substring(0,p.length-f),e[l+1][1]=m.substring(f),l++):(h>=p.length/2||h>=m.length/2)&&(e.splice(l,0,new t.Diff(0,p.substring(0,h))),e[l-1][0]=1,e[l-1][1]=m.substring(0,m.length-h),e[l+1][0]=n,e[l+1][1]=p.substring(h),l++),l++}l++}},t.prototype.diff_cleanupSemanticLossless=function(e){function n(e,n){if(!e||!n)return 6;var r=e.charAt(e.length-1),i=n.charAt(0),o=r.match(t.nonAlphaNumericRegex_),a=i.match(t.nonAlphaNumericRegex_),l=o&&r.match(t.whitespaceRegex_),s=a&&i.match(t.whitespaceRegex_),d=l&&r.match(t.linebreakRegex_),c=s&&i.match(t.linebreakRegex_),u=d&&e.match(t.blanklineEndRegex_),p=c&&n.match(t.blanklineStartRegex_);return u||p?5:d||c?4:o&&!l&&s?3:l||s?2:o||a?1:0}for(var r=1;r<e.length-1;){if(0==e[r-1][0]&&0==e[r+1][0]){var i=e[r-1][1],o=e[r][1],a=e[r+1][1],l=this.diff_commonSuffix(i,o);if(l){var s=o.substring(o.length-l);i=i.substring(0,i.length-l),o=s+o.substring(0,o.length-l),a=s+a}for(var d=i,c=o,u=a,p=n(i,o)+n(o,a);o.charAt(0)===a.charAt(0);){i+=o.charAt(0),o=o.substring(1)+a.charAt(0),a=a.substring(1);var m=n(i,o)+n(o,a);m>=p&&(p=m,d=i,c=o,u=a)}e[r-1][1]!=d&&(d?e[r-1][1]=d:(e.splice(r-1,1),r--),e[r][1]=c,u?e[r+1][1]=u:(e.splice(r+1,1),r--))}r++}},t.nonAlphaNumericRegex_=/[^a-zA-Z0-9]/,t.whitespaceRegex_=/\s/,t.linebreakRegex_=/[\r\n]/,t.blanklineEndRegex_=/\n\r?\n$/,t.blanklineStartRegex_=/^\r?\n\r?\n/,t.prototype.diff_cleanupEfficiency=function(e){for(var r=!1,i=[],o=0,a=null,l=0,s=!1,d=!1,c=!1,u=!1;l<e.length;)0==e[l][0]?(e[l][1].length<this.Diff_EditCost&&(c||u)?(i[o++]=l,s=c,d=u,a=e[l][1]):(o=0,a=null),c=u=!1):(e[l][0]==n?u=!0:c=!0,a&&(s&&d&&c&&u||a.length<this.Diff_EditCost/2&&s+d+c+u==3)&&(e.splice(i[o-1],0,new t.Diff(n,a)),e[i[o-1]+1][0]=1,o--,a=null,s&&d?(c=u=!0,o=0):(l=--o>0?i[o-1]:-1,c=u=!1),r=!0)),l++;r&&this.diff_cleanupMerge(e)},t.prototype.diff_cleanupMerge=function(e){e.push(new t.Diff(0,""));for(var r,i=0,o=0,a=0,l="",s="";i<e.length;)switch(e[i][0]){case 1:a++,s+=e[i][1],i++;break;case n:o++,l+=e[i][1],i++;break;case 0:o+a>1?(0!==o&&0!==a&&(0!==(r=this.diff_commonPrefix(s,l))&&(i-o-a>0&&0==e[i-o-a-1][0]?e[i-o-a-1][1]+=s.substring(0,r):(e.splice(0,0,new t.Diff(0,s.substring(0,r))),i++),s=s.substring(r),l=l.substring(r)),0!==(r=this.diff_commonSuffix(s,l))&&(e[i][1]=s.substring(s.length-r)+e[i][1],s=s.substring(0,s.length-r),l=l.substring(0,l.length-r))),i-=o+a,e.splice(i,o+a),l.length&&(e.splice(i,0,new t.Diff(n,l)),i++),s.length&&(e.splice(i,0,new t.Diff(1,s)),i++),i++):0!==i&&0==e[i-1][0]?(e[i-1][1]+=e[i][1],e.splice(i,1)):i++,a=0,o=0,l="",s=""}""===e[e.length-1][1]&&e.pop();var d=!1;for(i=1;i<e.length-1;)0==e[i-1][0]&&0==e[i+1][0]&&(e[i][1].substring(e[i][1].length-e[i-1][1].length)==e[i-1][1]?(e[i][1]=e[i-1][1]+e[i][1].substring(0,e[i][1].length-e[i-1][1].length),e[i+1][1]=e[i-1][1]+e[i+1][1],e.splice(i-1,1),d=!0):e[i][1].substring(0,e[i+1][1].length)==e[i+1][1]&&(e[i-1][1]+=e[i+1][1],e[i][1]=e[i][1].substring(e[i+1][1].length)+e[i+1][1],e.splice(i+1,1),d=!0)),i++;d&&this.diff_cleanupMerge(e)},t.prototype.diff_xIndex=function(e,t){var r,i=0,o=0,a=0,l=0;for(r=0;r<e.length&&(1!==e[r][0]&&(i+=e[r][1].length),e[r][0]!==n&&(o+=e[r][1].length),!(i>t));r++)a=i,l=o;return e.length!=r&&e[r][0]===n?l:l+(t-a)},t.prototype.diff_prettyHtml=function(e){for(var t=[],r=/&/g,i=/</g,o=/>/g,a=/\n/g,l=0;l<e.length;l++){var s=e[l][0],d=e[l][1].replace(r,"&amp;").replace(i,"&lt;").replace(o,"&gt;").replace(a,"&para;<br>");switch(s){case 1:t[l]='<ins style="background:#e6ffe6;">'+d+"</ins>";break;case n:t[l]='<del style="background:#ffe6e6;">'+d+"</del>";break;case 0:t[l]="<span>"+d+"</span>"}}return t.join("")},t.prototype.diff_text1=function(e){for(var t=[],n=0;n<e.length;n++)1!==e[n][0]&&(t[n]=e[n][1]);return t.join("")},t.prototype.diff_text2=function(e){for(var t=[],r=0;r<e.length;r++)e[r][0]!==n&&(t[r]=e[r][1]);return t.join("")},t.prototype.diff_levenshtein=function(e){for(var t=0,r=0,i=0,o=0;o<e.length;o++){var a=e[o][0],l=e[o][1];switch(a){case 1:r+=l.length;break;case n:i+=l.length;break;case 0:t+=Math.max(r,i),r=0,i=0}}return t+=Math.max(r,i)},t.prototype.diff_toDelta=function(e){for(var t=[],r=0;r<e.length;r++)switch(e[r][0]){case 1:t[r]="+"+encodeURI(e[r][1]);break;case n:t[r]="-"+e[r][1].length;break;case 0:t[r]="="+e[r][1].length}return t.join("\t").replace(/%20/g," ")},t.prototype.diff_fromDelta=function(e,r){for(var i=[],o=0,a=0,l=r.split(/\t/g),s=0;s<l.length;s++){var d=l[s].substring(1);switch(l[s].charAt(0)){case"+":try{i[o++]=new t.Diff(1,decodeURI(d))}catch(e){throw new Error("Illegal escape in diff_fromDelta: "+d)}break;case"-":case"=":var c=parseInt(d,10);if(isNaN(c)||c<0)throw new Error("Invalid number in diff_fromDelta: "+d);var u=e.substring(a,a+=c);"="==l[s].charAt(0)?i[o++]=new t.Diff(0,u):i[o++]=new t.Diff(n,u);break;default:if(l[s])throw new Error("Invalid diff operation in diff_fromDelta: "+l[s])}}if(a!=e.length)throw new Error("Delta length ("+a+") does not equal source text length ("+e.length+").");return i},t.prototype.match_main=function(e,t,n){if(null==e||null==t||null==n)throw new Error("Null input. (match_main)");return n=Math.max(0,Math.min(n,e.length)),e==t?0:e.length?e.substring(n,n+t.length)==t?n:this.match_bitap_(e,t,n):-1},t.prototype.match_bitap_=function(e,t,n){if(t.length>this.Match_MaxBits)throw new Error("Pattern too long for this browser.");var r=this.match_alphabet_(t),i=this;function o(e,r){var o=e/t.length,a=Math.abs(n-r);return i.Match_Distance?o+a/i.Match_Distance:a?1:o}var a=this.Match_Threshold,l=e.indexOf(t,n);-1!=l&&(a=Math.min(o(0,l),a),-1!=(l=e.lastIndexOf(t,n+t.length))&&(a=Math.min(o(0,l),a)));var s,d,c=1<<t.length-1;l=-1;for(var u,p=t.length+e.length,m=0;m<t.length;m++){for(s=0,d=p;s<d;)o(m,n+d)<=a?s=d:p=d,d=Math.floor((p-s)/2+s);p=d;var f=Math.max(1,n-d+1),h=Math.min(n+d,e.length)+t.length,v=Array(h+2);v[h+1]=(1<<m)-1;for(var g=h;g>=f;g--){var y=r[e.charAt(g-1)];if(v[g]=0===m?(v[g+1]<<1|1)&y:(v[g+1]<<1|1)&y|(u[g+1]|u[g])<<1|1|u[g+1],v[g]&c){var b=o(m,g-1);if(b<=a){if(a=b,!((l=g-1)>n))break;f=Math.max(1,2*n-l)}}}if(o(m+1,n)>a)break;u=v}return l},t.prototype.match_alphabet_=function(e){for(var t={},n=0;n<e.length;n++)t[e.charAt(n)]=0;for(n=0;n<e.length;n++)t[e.charAt(n)]|=1<<e.length-n-1;return t},t.prototype.patch_addContext_=function(e,n){if(0!=n.length){if(null===e.start2)throw Error("patch not initialized");for(var r=n.substring(e.start2,e.start2+e.length1),i=0;n.indexOf(r)!=n.lastIndexOf(r)&&r.length<this.Match_MaxBits-this.Patch_Margin-this.Patch_Margin;)i+=this.Patch_Margin,r=n.substring(e.start2-i,e.start2+e.length1+i);i+=this.Patch_Margin;var o=n.substring(e.start2-i,e.start2);o&&e.diffs.unshift(new t.Diff(0,o));var a=n.substring(e.start2+e.length1,e.start2+e.length1+i);a&&e.diffs.push(new t.Diff(0,a)),e.start1-=o.length,e.start2-=o.length,e.length1+=o.length+a.length,e.length2+=o.length+a.length}},t.prototype.patch_make=function(e,r,i){var o,a;if("string"==typeof e&&"string"==typeof r&&void 0===i)o=e,(a=this.diff_main(o,r,!0)).length>2&&(this.diff_cleanupSemantic(a),this.diff_cleanupEfficiency(a));else if(e&&"object"==typeof e&&void 0===r&&void 0===i)a=e,o=this.diff_text1(a);else if("string"==typeof e&&r&&"object"==typeof r&&void 0===i)o=e,a=r;else{if("string"!=typeof e||"string"!=typeof r||!i||"object"!=typeof i)throw new Error("Unknown call format to patch_make.");o=e,a=i}if(0===a.length)return[];for(var l=[],s=new t.patch_obj,d=0,c=0,u=0,p=o,m=o,f=0;f<a.length;f++){var h=a[f][0],v=a[f][1];switch(d||0===h||(s.start1=c,s.start2=u),h){case 1:s.diffs[d++]=a[f],s.length2+=v.length,m=m.substring(0,u)+v+m.substring(u);break;case n:s.length1+=v.length,s.diffs[d++]=a[f],m=m.substring(0,u)+m.substring(u+v.length);break;case 0:v.length<=2*this.Patch_Margin&&d&&a.length!=f+1?(s.diffs[d++]=a[f],s.length1+=v.length,s.length2+=v.length):v.length>=2*this.Patch_Margin&&d&&(this.patch_addContext_(s,p),l.push(s),s=new t.patch_obj,d=0,p=m,c=u)}1!==h&&(c+=v.length),h!==n&&(u+=v.length)}return d&&(this.patch_addContext_(s,p),l.push(s)),l},t.prototype.patch_deepCopy=function(e){for(var n=[],r=0;r<e.length;r++){var i=e[r],o=new t.patch_obj;o.diffs=[];for(var a=0;a<i.diffs.length;a++)o.diffs[a]=new t.Diff(i.diffs[a][0],i.diffs[a][1]);o.start1=i.start1,o.start2=i.start2,o.length1=i.length1,o.length2=i.length2,n[r]=o}return n},t.prototype.patch_apply=function(e,t){if(0==e.length)return[t,[]];e=this.patch_deepCopy(e);var r=this.patch_addPadding(e);t=r+t+r,this.patch_splitMax(e);for(var i=0,o=[],a=0;a<e.length;a++){var l,s,d=e[a].start2+i,c=this.diff_text1(e[a].diffs),u=-1;if(c.length>this.Match_MaxBits?-1!=(l=this.match_main(t,c.substring(0,this.Match_MaxBits),d))&&(-1==(u=this.match_main(t,c.substring(c.length-this.Match_MaxBits),d+c.length-this.Match_MaxBits))||l>=u)&&(l=-1):l=this.match_main(t,c,d),-1==l)o[a]=!1,i-=e[a].length2-e[a].length1;else if(o[a]=!0,i=l-d,c==(s=-1==u?t.substring(l,l+c.length):t.substring(l,u+this.Match_MaxBits)))t=t.substring(0,l)+this.diff_text2(e[a].diffs)+t.substring(l+c.length);else{var p=this.diff_main(c,s,!1);if(c.length>this.Match_MaxBits&&this.diff_levenshtein(p)/c.length>this.Patch_DeleteThreshold)o[a]=!1;else{this.diff_cleanupSemanticLossless(p);for(var m,f=0,h=0;h<e[a].diffs.length;h++){var v=e[a].diffs[h];0!==v[0]&&(m=this.diff_xIndex(p,f)),1===v[0]?t=t.substring(0,l+m)+v[1]+t.substring(l+m):v[0]===n&&(t=t.substring(0,l+m)+t.substring(l+this.diff_xIndex(p,f+v[1].length))),v[0]!==n&&(f+=v[1].length)}}}}return[t=t.substring(r.length,t.length-r.length),o]},t.prototype.patch_addPadding=function(e){for(var n=this.Patch_Margin,r="",i=1;i<=n;i++)r+=String.fromCharCode(i);for(i=0;i<e.length;i++)e[i].start1+=n,e[i].start2+=n;var o=e[0],a=o.diffs;if(0==a.length||0!=a[0][0])a.unshift(new t.Diff(0,r)),o.start1-=n,o.start2-=n,o.length1+=n,o.length2+=n;else if(n>a[0][1].length){var l=n-a[0][1].length;a[0][1]=r.substring(a[0][1].length)+a[0][1],o.start1-=l,o.start2-=l,o.length1+=l,o.length2+=l}if(0==(a=(o=e[e.length-1]).diffs).length||0!=a[a.length-1][0])a.push(new t.Diff(0,r)),o.length1+=n,o.length2+=n;else if(n>a[a.length-1][1].length){l=n-a[a.length-1][1].length;a[a.length-1][1]+=r.substring(0,l),o.length1+=l,o.length2+=l}return r},t.prototype.patch_splitMax=function(e){for(var r=this.Match_MaxBits,i=0;i<e.length;i++)if(!(e[i].length1<=r)){var o=e[i];e.splice(i--,1);for(var a=o.start1,l=o.start2,s="";0!==o.diffs.length;){var d=new t.patch_obj,c=!0;for(d.start1=a-s.length,d.start2=l-s.length,""!==s&&(d.length1=d.length2=s.length,d.diffs.push(new t.Diff(0,s)));0!==o.diffs.length&&d.length1<r-this.Patch_Margin;){var u=o.diffs[0][0],p=o.diffs[0][1];1===u?(d.length2+=p.length,l+=p.length,d.diffs.push(o.diffs.shift()),c=!1):u===n&&1==d.diffs.length&&0==d.diffs[0][0]&&p.length>2*r?(d.length1+=p.length,a+=p.length,c=!1,d.diffs.push(new t.Diff(u,p)),o.diffs.shift()):(p=p.substring(0,r-d.length1-this.Patch_Margin),d.length1+=p.length,a+=p.length,0===u?(d.length2+=p.length,l+=p.length):c=!1,d.diffs.push(new t.Diff(u,p)),p==o.diffs[0][1]?o.diffs.shift():o.diffs[0][1]=o.diffs[0][1].substring(p.length))}s=(s=this.diff_text2(d.diffs)).substring(s.length-this.Patch_Margin);var m=this.diff_text1(o.diffs).substring(0,this.Patch_Margin);""!==m&&(d.length1+=m.length,d.length2+=m.length,0!==d.diffs.length&&0===d.diffs[d.diffs.length-1][0]?d.diffs[d.diffs.length-1][1]+=m:d.diffs.push(new t.Diff(0,m))),c||e.splice(++i,0,d)}}},t.prototype.patch_toText=function(e){for(var t=[],n=0;n<e.length;n++)t[n]=e[n];return t.join("")},t.prototype.patch_fromText=function(e){var r=[];if(!e)return r;for(var i=e.split("\n"),o=0,a=/^@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@$/;o<i.length;){var l=i[o].match(a);if(!l)throw new Error("Invalid patch string: "+i[o]);var s=new t.patch_obj;for(r.push(s),s.start1=parseInt(l[1],10),""===l[2]?(s.start1--,s.length1=1):"0"==l[2]?s.length1=0:(s.start1--,s.length1=parseInt(l[2],10)),s.start2=parseInt(l[3],10),""===l[4]?(s.start2--,s.length2=1):"0"==l[4]?s.length2=0:(s.start2--,s.length2=parseInt(l[4],10)),o++;o<i.length;){var d=i[o].charAt(0);try{var c=decodeURI(i[o].substring(1))}catch(e){throw new Error("Illegal escape in patch_fromText: "+c)}if("-"==d)s.diffs.push(new t.Diff(n,c));else if("+"==d)s.diffs.push(new t.Diff(1,c));else if(" "==d)s.diffs.push(new t.Diff(0,c));else{if("@"==d)break;if(""!==d)throw new Error('Invalid patch mode "'+d+'" in: '+c)}o++}}return r},(t.patch_obj=function(){this.diffs=[],this.start1=null,this.start2=null,this.length1=0,this.length2=0}).prototype.toString=function(){for(var e,t=["@@ -"+(0===this.length1?this.start1+",0":1==this.length1?this.start1+1:this.start1+1+","+this.length1)+" +"+(0===this.length2?this.start2+",0":1==this.length2?this.start2+1:this.start2+1+","+this.length2)+" @@\n"],r=0;r<this.diffs.length;r++){switch(this.diffs[r][0]){case 1:e="+";break;case n:e="-";break;case 0:e=" "}t[r+1]=e+encodeURI(this.diffs[r][1])+"\n"}return t.join("").replace(/%20/g," ")},e.exports=t,e.exports.diff_match_patch=t,e.exports.DIFF_DELETE=n,e.exports.DIFF_INSERT=1,e.exports.DIFF_EQUAL=0},157:()=>{},857:(e,t,n)=>{"use strict";n.d(t,{default:()=>N});var r=n(369),i=n(46),o=n(726),a=n(23),l=n(383),s=n(890),d=n(93),c=function(e){void 0===e&&(e=document);var t=function(e){var t=document.createElement("img");t.src=e.getAttribute("data-src"),t.addEventListener("load",(function(){e.getAttribute("style")||e.getAttribute("class")||e.getAttribute("width")||e.getAttribute("height")||t.naturalHeight>t.naturalWidth&&t.naturalWidth/t.naturalHeight<document.querySelector(".vditor-reset").clientWidth/(window.innerHeight-40)&&t.naturalHeight>window.innerHeight-40&&(e.style.height=window.innerHeight-40+"px"),e.src=t.src})),e.removeAttribute("data-src")};if(!("IntersectionObserver"in window))return e.querySelectorAll("img").forEach((function(e){e.getAttribute("data-src")&&t(e)})),!1;window.vditorImageIntersectionObserver?(window.vditorImageIntersectionObserver.disconnect(),e.querySelectorAll("img").forEach((function(e){window.vditorImageIntersectionObserver.observe(e)}))):(window.vditorImageIntersectionObserver=new IntersectionObserver((function(e){e.forEach((function(e){(void 0===e.isIntersecting?0!==e.intersectionRatio:e.isIntersecting)&&e.target.getAttribute("data-src")&&t(e.target)}))})),e.querySelectorAll("img").forEach((function(e){window.vditorImageIntersectionObserver.observe(e)})))},u=n(323),p=n(207),m=n(765),f=n(894),h=n(198),v=n(583),g=n(260),y=n(958),b=n(228),w=n(713),E=n(224),k=n(792),S=n(187),C=function(e,t){if(void 0===t&&(t="zh_CN"),"undefined"!=typeof speechSynthesis&&"undefined"!=typeof SpeechSynthesisUtterance){var n='<svg><use xlink:href="#vditor-icon-play"></use></svg>',r='<svg><use xlink:href="#vditor-icon-pause"></use></svg>';document.getElementById("vditorIconScript")||(n='<svg viewBox="0 0 32 32"><path d="M3.436 0l25.128 16-25.128 16v-32z"></path></svg>',r='<svg viewBox="0 0 32 32"><path d="M20.617 0h9.128v32h-9.128v-32zM2.255 32v-32h9.128v32h-9.128z"></path></svg>');var i=document.querySelector(".vditor-speech");if(!i){(i=document.createElement("div")).className="vditor-speech",document.body.insertAdjacentElement("beforeend",i);var o=function(){var e,n;return speechSynthesis.getVoices().forEach((function(r){r.lang===t.replace("_","-")&&(e=r),r.default&&(n=r)})),e||(e=n),e};void 0!==speechSynthesis.onvoiceschanged&&(speechSynthesis.onvoiceschanged=o);var a=o();i.onclick=function(){if("vditor-speech"===i.className){var e=new SpeechSynthesisUtterance(i.getAttribute("data-text"));e.voice=a,e.onend=function(){i.className="vditor-speech",speechSynthesis.cancel(),i.innerHTML=n},speechSynthesis.speak(e),i.className="vditor-speech vditor-speech--current",i.innerHTML=r}else speechSynthesis.speaking&&(speechSynthesis.paused?(speechSynthesis.resume(),i.innerHTML=r):(speechSynthesis.pause(),i.innerHTML=n));(0,S.Hc)(window.vditorSpeechRange)},document.body.addEventListener("click",(function(){""===getSelection().toString().trim()&&"block"===i.style.display&&(i.className="vditor-speech",speechSynthesis.cancel(),i.style.display="none")}))}e.addEventListener("mouseup",(function(e){var t=getSelection().toString().trim();if(speechSynthesis.cancel(),""!==getSelection().toString().trim()){window.vditorSpeechRange=getSelection().getRangeAt(0).cloneRange();var r=getSelection().getRangeAt(0).getBoundingClientRect();i.innerHTML=n,i.style.display="block",i.style.top=r.top+r.height+document.querySelector("html").scrollTop-20+"px",i.style.left=e.screenX+2+"px",i.setAttribute("data-text",t)}else"block"===i.style.display&&(i.className="vditor-speech",i.style.display="none")}))}},L=function(e,t,n,r){return new(n||(n=Promise))((function(i,o){function a(e){try{s(r.next(e))}catch(e){o(e)}}function l(e){try{s(r.throw(e))}catch(e){o(e)}}function s(e){var t;e.done?i(e.value):(t=e.value,t instanceof n?t:new n((function(e){e(t)}))).then(a,l)}s((r=r.apply(e,t||[])).next())}))},T=function(e,t){var n,r,i,o,a={label:0,sent:function(){if(1&i[0])throw i[1];return i[1]},trys:[],ops:[]};return o={next:l(0),throw:l(1),return:l(2)},"function"==typeof Symbol&&(o[Symbol.iterator]=function(){return this}),o;function l(o){return function(l){return function(o){if(n)throw new TypeError("Generator is already executing.");for(;a;)try{if(n=1,r&&(i=2&o[0]?r.return:o[0]?r.throw||((i=r.return)&&i.call(r),0):r.next)&&!(i=i.call(r,o[1])).done)return i;switch(r=0,i&&(o=[2&o[0],i.value]),o[0]){case 0:case 1:i=o;break;case 4:return a.label++,{value:o[1],done:!1};case 5:a.label++,r=o[1],o=[0];continue;case 7:o=a.ops.pop(),a.trys.pop();continue;default:if(!(i=a.trys,(i=i.length>0&&i[i.length-1])||6!==o[0]&&2!==o[0])){a=0;continue}if(3===o[0]&&(!i||o[1]>i[0]&&o[1]<i[3])){a.label=o[1];break}if(6===o[0]&&a.label<i[1]){a.label=i[1],i=o;break}if(i&&a.label<i[2]){a.label=i[2],a.ops.push(o);break}i[2]&&a.ops.pop(),a.trys.pop();continue}o=t.call(e,a)}catch(e){o=[6,e],r=0}finally{n=i=0}if(5&o[0])throw o[1];return{value:o[0]?o[1]:void 0,done:!0}}([o,l])}}},M=function(e){var t={anchor:0,cdn:g.g.CDN,customEmoji:{},emojiPath:(e&&e.emojiPath||g.g.CDN)+"/dist/images/emoji",hljs:g.g.HLJS_OPTIONS,icon:"ant",lang:"zh_CN",markdown:g.g.MARKDOWN_OPTIONS,math:g.g.MATH_OPTIONS,mode:"light",speech:{enable:!1},theme:g.g.THEME_OPTIONS};return(0,E.T)(t,e)},A=function(e,t){var n=M(t);return(0,b.G)(n.cdn+"/dist/js/lute/lute.min.js","vditorLuteScript").then((function(){var r=(0,k.X)({autoSpace:n.markdown.autoSpace,codeBlockPreview:n.markdown.codeBlockPreview,emojiSite:n.emojiPath,emojis:n.customEmoji,fixTermTypo:n.markdown.fixTermTypo,footnotes:n.markdown.footnotes,headingAnchor:0!==n.anchor,inlineMathDigit:n.math.inlineDigit,lazyLoadImage:n.lazyLoadImage,linkBase:n.markdown.linkBase,linkPrefix:n.markdown.linkPrefix,listStyle:n.markdown.listStyle,mark:n.markdown.mark,mathBlockPreview:n.markdown.mathBlockPreview,paragraphBeginningSpace:n.markdown.paragraphBeginningSpace,sanitize:n.markdown.sanitize,toc:n.markdown.toc});return(null==t?void 0:t.renderers)&&r.SetJSRenderers({renderers:{Md2HTML:t.renderers}}),r.SetHeadingID(!0),r.Md2HTML(e)}))},_=function(e,t,n){return L(void 0,void 0,void 0,(function(){var i,h,g;return T(this,(function(E){switch(E.label){case 0:return i=M(n),[4,A(t,i)];case 1:if(h=E.sent(),i.transform&&(h=i.transform(h)),e.innerHTML=h,e.classList.add("vditor-reset"),i.i18n)return[3,5];if(["en_US","ja_JP","ko_KR","ru_RU","zh_CN","zh_TW"].includes(i.lang))return[3,2];throw new Error("options.lang error, see https://ld246.com/article/1549638745630#options");case 2:return g="vditorI18nScript"+i.lang,document.querySelectorAll('head script[id^="vditorI18nScript"]').forEach((function(e){e.id!==g&&document.head.removeChild(e)})),[4,(0,b.G)(i.cdn+"/dist/js/i18n/"+i.lang+".js",g)];case 3:E.sent(),E.label=4;case 4:return[3,6];case 5:window.VditorI18n=i.i18n,E.label=6;case 6:return i.icon?[4,(0,b.G)(i.cdn+"/dist/js/icons/"+i.icon+".js","vditorIconScript")]:[3,8];case 7:E.sent(),E.label=8;case 8:return(0,y.Z)(i.theme.current,i.theme.path),1===i.anchor&&e.classList.add("vditor-reset--anchor"),(0,a.O)(e),(0,d.s)(i.hljs,e,i.cdn),(0,u.H)(e,{cdn:i.cdn,math:i.math}),(0,m.i)(e,i.cdn,i.mode),(0,l.P)(e,i.cdn),(0,s.v)(e,i.cdn),(0,o.p)(e,i.cdn,i.mode),(0,f.P)(e,i.cdn,i.mode),(0,v.B)(e,i.cdn),(0,r.Q)(e,i.cdn),(0,p.Y)(e),i.speech.enable&&C(e),0!==i.anchor&&(k=i.anchor,document.querySelectorAll(".vditor-anchor").forEach((function(e){1===k&&e.classList.add("vditor-anchor--left"),e.onclick=function(){var t=e.getAttribute("href").substr(1),n=document.getElementById("vditorAnchor-"+t).offsetTop;document.querySelector("html").scrollTop=n}})),window.onhashchange=function(){var e=document.getElementById("vditorAnchor-"+decodeURIComponent(window.location.hash.substr(1)));e&&(document.querySelector("html").scrollTop=e.offsetTop)}),i.after&&i.after(),i.lazyLoadImage&&c(e),e.addEventListener("click",(function(t){var n=(0,w.lG)(t.target,"SPAN");if(n&&(0,w.fb)(n,"vditor-toc")){var r=e.querySelector("#"+n.getAttribute("data-target-id"));r&&window.scrollTo(window.scrollX,r.offsetTop)}else;})),[2]}var k}))}))},x=n(264),H=n(968);const N=function(){function e(){}return e.adapterRender=i,e.previewImage=x.E,e.codeRender=a.O,e.graphvizRender=s.v,e.highlightRender=d.s,e.mathRender=u.H,e.mermaidRender=m.i,e.flowchartRender=l.P,e.chartRender=o.p,e.abcRender=r.Q,e.mindmapRender=f.P,e.plantumlRender=v.B,e.outlineRender=h.k,e.mediaRender=p.Y,e.speechRender=C,e.lazyLoadImageRender=c,e.md2html=A,e.preview=_,e.setCodeTheme=H.Y,e.setContentTheme=y.Z,e}()},260:(e,t,n)=>{"use strict";n.d(t,{H:()=>r,g:()=>i});var r="3.8.11",i=function(){function e(){}return e.ZWSP="​",e.DROP_EDITOR="application/editor",e.MOBILE_WIDTH=520,e.CLASS_MENU_DISABLED="vditor-menu--disabled",e.EDIT_TOOLBARS=["emoji","headings","bold","italic","strike","link","list","ordered-list","outdent","indent","check","line","quote","code","inline-code","insert-after","insert-before","upload","record","table"],e.CODE_THEME=["abap","algol","algol_nu","arduino","autumn","borland","bw","colorful","dracula","emacs","friendly","fruity","github","igor","lovelace","manni","monokai","monokailight","murphy","native","paraiso-dark","paraiso-light","pastie","perldoc","pygments","rainbow_dash","rrt","solarized-dark","solarized-dark256","solarized-light","swapoff","tango","trac","vim","vs","xcode","ant-design"],e.CODE_LANGUAGES=["mermaid","echarts","mindmap","plantuml","abc","graphviz","flowchart","apache","js","ts","html","properties","apache","bash","c","csharp","cpp","css","coffeescript","diff","go","xml","http","json","java","javascript","kotlin","less","lua","makefile","markdown","nginx","objectivec","php","php-template","perl","plaintext","python","python-repl","r","ruby","rust","scss","sql","shell","swift","ini","typescript","vbnet","yaml","ada","clojure","dart","erb","fortran","gradle","haskell","julia","julia-repl","lisp","matlab","pgsql","powershell","sql_more","stata","cmake","mathematica"],e.CDN="https://cdn.jsdelivr.net/npm/vditor@3.8.11",e.MARKDOWN_OPTIONS={autoSpace:!1,codeBlockPreview:!0,fixTermTypo:!1,footnotes:!0,linkBase:"",linkPrefix:"",listStyle:!1,mark:!1,mathBlockPreview:!0,paragraphBeginningSpace:!1,sanitize:!0,toc:!1},e.HLJS_OPTIONS={enable:!0,lineNumber:!1,style:"github"},e.MATH_OPTIONS={engine:"KaTeX",inlineDigit:!1,macros:{}},e.THEME_OPTIONS={current:"light",list:{"ant-design":"Ant Design",dark:"Dark",light:"Light",wechat:"WeChat"},path:e.CDN+"/dist/css/content-theme"},e}()},369:(e,t,n)=>{"use strict";n.d(t,{Q:()=>a});var r=n(260),i=n(228),o=n(46),a=function(e,t){void 0===e&&(e=document),void 0===t&&(t=r.g.CDN);var n=o.abcRenderAdapter.getElements(e);n.length>0&&(0,i.G)(t+"/dist/js/abcjs/abcjs_basic.min.js","vditorAbcjsScript").then((function(){n.forEach((function(e){e.parentElement.classList.contains("vditor-wysiwyg__pre")||e.parentElement.classList.contains("vditor-ir__marker--pre")||"true"!==e.getAttribute("data-processed")&&(ABCJS.renderAbc(e,o.abcRenderAdapter.getCode(e).trim()),e.style.overflowX="auto",e.setAttribute("data-processed","true"))}))}))}},46:(e,t,n)=>{"use strict";n.r(t),n.d(t,{mathRenderAdapter:()=>r,mermaidRenderAdapter:()=>i,mindmapRenderAdapter:()=>o,chartRenderAdapter:()=>a,abcRenderAdapter:()=>l,graphvizRenderAdapter:()=>s,flowchartRenderAdapter:()=>d,plantumlRenderAdapter:()=>c});var r={getCode:function(e){return e.textContent},getElements:function(e){return e.querySelectorAll(".language-math")}},i={getCode:function(e){return e.textContent},getElements:function(e){return e.querySelectorAll(".language-mermaid")}},o={getCode:function(e){return e.getAttribute("data-code")},getElements:function(e){return e.querySelectorAll(".language-mindmap")}},a={getCode:function(e){return e.innerText},getElements:function(e){return e.querySelectorAll(".language-echarts")}},l={getCode:function(e){return e.textContent},getElements:function(e){return e.querySelectorAll(".language-abc")}},s={getCode:function(e){return e.textContent},getElements:function(e){return e.querySelectorAll(".language-graphviz")}},d={getCode:function(e){return e.textContent},getElements:function(e){return e.querySelectorAll(".language-flowchart")}},c={getCode:function(e){return e.textContent},getElements:function(e){return e.querySelectorAll(".language-plantuml")}}},726:(e,t,n)=>{"use strict";n.d(t,{p:()=>a});var r=n(260),i=n(228),o=n(46),a=function(e,t,n){void 0===e&&(e=document),void 0===t&&(t=r.g.CDN);var a=o.chartRenderAdapter.getElements(e);a.length>0&&(0,i.G)(t+"/dist/js/echarts/echarts.min.js","vditorEchartsScript").then((function(){a.forEach((function(e){if(!e.parentElement.classList.contains("vditor-wysiwyg__pre")&&!e.parentElement.classList.contains("vditor-ir__marker--pre")){var t=o.chartRenderAdapter.getCode(e).trim();if(t)try{if("true"===e.getAttribute("data-processed"))return;var r=JSON.parse(t);echarts.init(e,"dark"===n?"dark":void 0).setOption(r),e.setAttribute("data-processed","true")}catch(t){e.className="vditor-reset--error",e.innerHTML="echarts render error: <br>"+t}}}))}))}},23:(e,t,n)=>{"use strict";n.d(t,{O:()=>i});var r=n(769),i=function(e){e.querySelectorAll("pre > code").forEach((function(t,n){var i,o,a;if(!t.parentElement.classList.contains("vditor-wysiwyg__pre")&&!t.parentElement.classList.contains("vditor-ir__marker--pre")&&!(t.classList.contains("language-mermaid")||t.classList.contains("language-flowchart")||t.classList.contains("language-echarts")||t.classList.contains("language-mindmap")||t.classList.contains("language-plantuml")||t.classList.contains("language-abc")||t.classList.contains("language-graphviz")||t.classList.contains("language-math")||t.style.maxHeight.indexOf("px")>-1||e.classList.contains("vditor-preview")&&n>5)){var l=t.innerText;if(t.classList.contains("highlight-chroma")){var s=document.createElement("code");s.innerHTML=t.innerHTML,s.querySelectorAll(".highlight-ln").forEach((function(e){e.remove()})),l=s.innerText}var d='<svg><use xlink:href="#vditor-icon-copy"></use></svg>';document.getElementById("vditorIconScript")||(d='<svg viewBox="0 0 32 32"><path d="M22.545-0h-17.455c-1.6 0-2.909 1.309-2.909 2.909v20.364h2.909v-20.364h17.455v-2.909zM26.909 5.818h-16c-1.6 0-2.909 1.309-2.909 2.909v20.364c0 1.6 1.309 2.909 2.909 2.909h16c1.6 0 2.909-1.309 2.909-2.909v-20.364c0-1.6-1.309-2.909-2.909-2.909zM26.909 29.091h-16v-20.364h16v20.364z"></path></svg>');var c=document.createElement("div");c.className="vditor-copy",c.innerHTML='<span aria-label="'+((null===(i=window.VditorI18n)||void 0===i?void 0:i.copy)||"复制")+"\"\nonmouseover=\"this.setAttribute('aria-label', '"+((null===(o=window.VditorI18n)||void 0===o?void 0:o.copy)||"复制")+"')\"\nclass=\"vditor-tooltipped vditor-tooltipped__w\"\nonclick=\"this.previousElementSibling.select();document.execCommand('copy');this.setAttribute('aria-label', '"+((null===(a=window.VditorI18n)||void 0===a?void 0:a.copy)||"已复制")+"')\">"+d+"</span>";var u=document.createElement("textarea");u.value=(0,r.X)(l),c.insertAdjacentElement("afterbegin",u),t.before(c),t.style.maxHeight=window.outerHeight-40+"px"}}))}},383:(e,t,n)=>{"use strict";n.d(t,{P:()=>a});var r=n(260),i=n(228),o=n(46),a=function(e,t){void 0===t&&(t=r.g.CDN);var n=o.flowchartRenderAdapter.getElements(e);0!==n.length&&(0,i.G)(t+"/dist/js/flowchart.js/flowchart.min.js","vditorFlowchartScript").then((function(){n.forEach((function(e){if("true"!==e.getAttribute("data-processed")){var t=flowchart.parse(o.flowchartRenderAdapter.getCode(e));e.innerHTML="",t.drawSVG(e),e.setAttribute("data-processed","true")}}))}))}},890:(e,t,n)=>{"use strict";n.d(t,{v:()=>a});var r=n(260),i=n(228),o=n(46),a=function(e,t){void 0===t&&(t=r.g.CDN);var n=o.graphvizRenderAdapter.getElements(e);0!==n.length&&(0,i.G)(t+"/dist/js/graphviz/viz.js","vditorGraphVizScript").then((function(){n.forEach((function(e){var t=o.graphvizRenderAdapter.getCode(e);if(!e.parentElement.classList.contains("vditor-wysiwyg__pre")&&!e.parentElement.classList.contains("vditor-ir__marker--pre")&&"true"!==e.getAttribute("data-processed")&&""!==t.trim()){try{var n=new Blob(["importScripts('"+document.getElementById("vditorGraphVizScript").src.replace("viz.js","full.render.js")+"');"],{type:"application/javascript"}),r=(window.URL||window.webkitURL).createObjectURL(n),i=new Worker(r);new Viz({worker:i}).renderSVGElement(t).then((function(t){e.innerHTML=t.outerHTML})).catch((function(t){e.innerHTML="graphviz render error: <br>"+t,e.className="vditor-reset--error"}))}catch(e){console.error("graphviz error",e)}e.setAttribute("data-processed","true")}}))}))}},93:(e,t,n)=>{"use strict";n.d(t,{s:()=>a});var r=n(260),i=n(228),o=n(946),a=function(e,t,n){void 0===t&&(t=document),void 0===n&&(n=r.g.CDN);var a=e.style;r.g.CODE_THEME.includes(a)||(a="github");var l=document.getElementById("vditorHljsStyle"),s=n+"/dist/js/highlight.js/styles/"+a+".css";(l&&l.href!==s&&l.remove(),(0,o.c)(n+"/dist/js/highlight.js/styles/"+a+".css","vditorHljsStyle"),!1!==e.enable)&&(0!==t.querySelectorAll("pre > code").length&&(0,i.G)(n+"/dist/js/highlight.js/highlight.pack.js","vditorHljsScript").then((function(){t.querySelectorAll("pre > code").forEach((function(t){if(!t.parentElement.classList.contains("vditor-ir__marker--pre")&&!t.parentElement.classList.contains("vditor-wysiwyg__pre")&&!(t.classList.contains("language-mermaid")||t.classList.contains("language-flowchart")||t.classList.contains("language-echarts")||t.classList.contains("language-mindmap")||t.classList.contains("language-plantuml")||t.classList.contains("language-abc")||t.classList.contains("language-graphviz")||t.classList.contains("language-math"))&&(hljs.highlightElement(t),e.lineNumber)){t.classList.add("vditor-linenumber");var n=t.querySelector(".vditor-linenumber__temp");n||((n=document.createElement("div")).className="vditor-linenumber__temp",t.insertAdjacentElement("beforeend",n));var r=getComputedStyle(t).whiteSpace,i=!1;"pre-wrap"!==r&&"pre-line"!==r||(i=!0);var o="",a=t.textContent.split(/\r\n|\r|\n/g);a.pop(),a.map((function(e){var t="";i&&(n.textContent=e||"\n",t=' style="height:'+n.getBoundingClientRect().height+'px"'),o+="<span"+t+"></span>"})),n.style.display="none",o='<span class="vditor-linenumber__rows">'+o+"</span>",t.insertAdjacentHTML("beforeend",o)}}))})))}},323:(e,t,n)=>{"use strict";n.d(t,{H:()=>s});var r=n(260),i=n(228),o=n(946),a=n(769),l=n(46),s=function(e,t){var n=l.mathRenderAdapter.getElements(e);if(0!==n.length){var s={cdn:r.g.CDN,math:{engine:"KaTeX",inlineDigit:!1,macros:{}}};if(t&&t.math&&(t.math=Object.assign({},s.math,t.math)),"KaTeX"===(t=Object.assign({},s,t)).math.engine)(0,o.c)(t.cdn+"/dist/js/katex/katex.min.css","vditorKatexStyle"),(0,i.G)(t.cdn+"/dist/js/katex/katex.min.js","vditorKatexScript").then((function(){(0,i.G)(t.cdn+"/dist/js/katex/mhchem.min.js","vditorKatexChemScript").then((function(){n.forEach((function(e){if(!e.parentElement.classList.contains("vditor-wysiwyg__pre")&&!e.parentElement.classList.contains("vditor-ir__marker--pre")&&!e.getAttribute("data-math")){var t=(0,a.X)(l.mathRenderAdapter.getCode(e));e.setAttribute("data-math",t);try{e.innerHTML=katex.renderToString(t,{displayMode:"DIV"===e.tagName,output:"html"})}catch(t){e.innerHTML=t.message,e.className="language-math vditor-reset--error"}e.addEventListener("copy",(function(e){e.stopPropagation(),e.preventDefault();var t=e.currentTarget.closest(".language-math");e.clipboardData.setData("text/html",t.innerHTML),e.clipboardData.setData("text/plain",t.getAttribute("data-math"))}))}}))}))}));else if("MathJax"===t.math.engine){window.MathJax||(window.MathJax={loader:{paths:{mathjax:t.cdn+"/dist/js/mathjax"}},startup:{typeset:!1},tex:{macros:t.math.macros}}),(0,i.J)(t.cdn+"/dist/js/mathjax/tex-svg-full.js","protyleMathJaxScript");var d=function(e,t){var n=(0,a.X)(e.textContent).trim(),r=window.MathJax.getMetricsFor(e);r.display="DIV"===e.tagName,window.MathJax.tex2svgPromise(n,r).then((function(r){e.innerHTML="",e.setAttribute("data-math",n),e.append(r),window.MathJax.startup.document.clear(),window.MathJax.startup.document.updateDocument();var i=r.querySelector('[data-mml-node="merror"]');i&&""!==i.textContent.trim()&&(e.innerHTML=i.textContent.trim(),e.className="vditor-reset--error"),t&&t()}))};window.MathJax.startup.promise.then((function(){for(var e=[],t=function(t){var r=n[t];r.parentElement.classList.contains("vditor-wysiwyg__pre")||r.parentElement.classList.contains("vditor-ir__marker--pre")||r.getAttribute("data-math")||!(0,a.X)(r.textContent).trim()||e.push((function(e){t===n.length-1?d(r):d(r,e)}))},r=0;r<n.length;r++)t(r);!function(e){if(0!==e.length){var t=0,n=e[e.length-1],r=function(){var i=e[t++];i===n?i():i(r)};r()}}(e)}))}}}},207:(e,t,n)=>{"use strict";n.d(t,{Y:()=>r});var r=function(e){e&&e.querySelectorAll("a").forEach((function(e){var t=e.getAttribute("href");t&&(t.match(/^.+.(mp4|m4v|ogg|ogv|webm)$/)?function(e,t){e.insertAdjacentHTML("afterend",'<video controls="controls" src="'+t+'"></video>'),e.remove()}(e,t):t.match(/^.+.(mp3|wav|flac)$/)?function(e,t){e.insertAdjacentHTML("afterend",'<audio controls="controls" src="'+t+'"></audio>'),e.remove()}(e,t):function(e,t){var n=t.match(/\/\/(?:www\.)?(?:youtu\.be\/|youtube\.com\/(?:embed\/|v\/|watch\?v=|watch\?.+&v=))([\w|-]{11})(?:(?:[\?&]t=)(\S+))?/),r=t.match(/\/\/v\.youku\.com\/v_show\/id_(\w+)=*\.html/),i=t.match(/\/\/v\.qq\.com\/x\/cover\/.*\/([^\/]+)\.html\??.*/),o=t.match(/(?:www\.|\/\/)coub\.com\/view\/(\w+)/),a=t.match(/(?:www\.|\/\/)facebook\.com\/([^\/]+)\/videos\/([0-9]+)/),l=t.match(/.+dailymotion.com\/(video|hub)\/(\w+)\?/),s=t.match(/(?:www\.|\/\/)bilibili\.com\/video\/(\w+)/),d=t.match(/(?:www\.|\/\/)ted\.com\/talks\/(\w+)/);n&&11===n[1].length?(e.insertAdjacentHTML("afterend",'<iframe class="iframe__video" src="//www.youtube.com/embed/'+n[1]+(n[2]?"?start="+n[2]:"")+'"></iframe>'),e.remove()):r&&r[1]?(e.insertAdjacentHTML("afterend",'<iframe class="iframe__video" src="//player.youku.com/embed/'+r[1]+'"></iframe>'),e.remove()):i&&i[1]?(e.insertAdjacentHTML("afterend",'<iframe class="iframe__video" src="https://v.qq.com/txp/iframe/player.html?vid='+i[1]+'"></iframe>'),e.remove()):o&&o[1]?(e.insertAdjacentHTML("afterend",'<iframe class="iframe__video"\n src="//coub.com/embed/'+o[1]+'?muted=false&autostart=false&originalSize=true&startWithHD=true"></iframe>'),e.remove()):a&&a[0]?(e.insertAdjacentHTML("afterend",'<iframe class="iframe__video"\n src="https://www.facebook.com/plugins/video.php?href='+encodeURIComponent(a[0])+'"></iframe>'),e.remove()):l&&l[2]?(e.insertAdjacentHTML("afterend",'<iframe class="iframe__video"\n src="https://www.dailymotion.com/embed/video/'+l[2]+'"></iframe>'),e.remove()):s&&s[1]?(e.insertAdjacentHTML("afterend",'<iframe class="iframe__video"\n src="//player.bilibili.com/player.html?bvid='+s[1]+'"></iframe>'),e.remove()):d&&d[1]&&(e.insertAdjacentHTML("afterend",'<iframe class="iframe__video" src="//embed.ted.com/talks/'+d[1]+'"></iframe>'),e.remove())}(e,t))}))}},765:(e,t,n)=>{"use strict";n.d(t,{i:()=>a});var r=n(260),i=n(228),o=n(46),a=function(e,t,n){void 0===t&&(t=r.g.CDN);var a=o.mermaidRenderAdapter.getElements(e);0!==a.length&&(0,i.G)(t+"/dist/js/mermaid/mermaid.min.js","vditorMermaidScript").then((function(){var e={altFontFamily:"sans-serif",flowchart:{htmlLabels:!0,useMaxWidth:!0},fontFamily:"sans-serif",gantt:{leftPadding:75,rightPadding:20},securityLevel:"loose",sequence:{boxMargin:8,diagramMarginX:8,diagramMarginY:8,useMaxWidth:!0},startOnLoad:!1};"dark"===n&&(e.theme="dark",e.themeVariables={activationBkgColor:"hsl(180, 1.5873015873%, 28.3529411765%)",activationBorderColor:"#81B1DB",activeTaskBkgColor:"#81B1DB",activeTaskBorderColor:"#ffffff",actorBkg:"#1f2020",actorBorder:"#81B1DB",actorLineColor:"lightgrey",actorTextColor:"lightgrey",altBackground:"hsl(0, 0%, 40%)",altSectionBkgColor:"#333",arrowheadColor:"lightgrey",background:"#333",border1:"#81B1DB",border2:"rgba(255, 255, 255, 0.25)",classText:"#e0dfdf",clusterBkg:"hsl(180, 1.5873015873%, 28.3529411765%)",clusterBorder:"rgba(255, 255, 255, 0.25)",critBkgColor:"#E83737",critBorderColor:"#E83737",darkTextColor:"hsl(28.5714285714, 17.3553719008%, 86.2745098039%)",defaultLinkColor:"lightgrey",doneTaskBkgColor:"lightgrey",doneTaskBorderColor:"grey",edgeLabelBackground:"hsl(0, 0%, 34.4117647059%)",errorBkgColor:"#a44141",errorTextColor:"#ddd",fillType0:"#1f2020",fillType1:"hsl(180, 1.5873015873%, 28.3529411765%)",fillType2:"hsl(244, 1.5873015873%, 12.3529411765%)",fillType3:"hsl(244, 1.5873015873%, 28.3529411765%)",fillType4:"hsl(116, 1.5873015873%, 12.3529411765%)",fillType5:"hsl(116, 1.5873015873%, 28.3529411765%)",fillType6:"hsl(308, 1.5873015873%, 12.3529411765%)",fillType7:"hsl(308, 1.5873015873%, 28.3529411765%)",fontFamily:'"trebuchet ms", verdana, arial',fontSize:"16px",gridColor:"lightgrey",labelBackground:"#181818",labelBoxBkgColor:"#1f2020",labelBoxBorderColor:"#81B1DB",labelColor:"#ccc",labelTextColor:"lightgrey",lineColor:"lightgrey",loopTextColor:"lightgrey",mainBkg:"#1f2020",mainContrastColor:"lightgrey",nodeBkg:"#1f2020",nodeBorder:"#81B1DB",noteBkgColor:"#fff5ad",noteBorderColor:"rgba(255, 255, 255, 0.25)",noteTextColor:"#1f2020",primaryBorderColor:"hsl(180, 0%, 2.3529411765%)",primaryColor:"#1f2020",primaryTextColor:"#e0dfdf",secondBkg:"hsl(180, 1.5873015873%, 28.3529411765%)",secondaryBorderColor:"hsl(180, 0%, 18.3529411765%)",secondaryColor:"hsl(180, 1.5873015873%, 28.3529411765%)",secondaryTextColor:"rgb(183.8476190475, 181.5523809523, 181.5523809523)",sectionBkgColor:"hsl(52.9411764706, 28.813559322%, 58.431372549%)",sectionBkgColor2:"#EAE8D9",sequenceNumberColor:"black",signalColor:"lightgrey",signalTextColor:"lightgrey",taskBkgColor:"hsl(180, 1.5873015873%, 35.3529411765%)",taskBorderColor:"#ffffff",taskTextClickableColor:"#003163",taskTextColor:"hsl(28.5714285714, 17.3553719008%, 86.2745098039%)",taskTextDarkColor:"hsl(28.5714285714, 17.3553719008%, 86.2745098039%)",taskTextLightColor:"lightgrey",taskTextOutsideColor:"lightgrey",tertiaryBorderColor:"hsl(20, 0%, 2.3529411765%)",tertiaryColor:"hsl(20, 1.5873015873%, 12.3529411765%)",tertiaryTextColor:"rgb(222.9999999999, 223.6666666666, 223.9999999999)",textColor:"#ccc",titleColor:"#F9FFFE",todayLineColor:"#DB5757"}),mermaid.initialize(e),a.forEach((function(e){var t=o.mermaidRenderAdapter.getCode(e);"true"!==e.getAttribute("data-processed")&&""!==t.trim()&&(mermaid.init(void 0,e),e.setAttribute("data-processed","true"))}))}))}},894:(e,t,n)=>{"use strict";n.d(t,{P:()=>a});var r=n(260),i=n(228),o=n(46),a=function(e,t,n){void 0===e&&(e=document),void 0===t&&(t=r.g.CDN);var a=o.mindmapRenderAdapter.getElements(e);a.length>0&&(0,i.G)(t+"/dist/js/echarts/echarts.min.js","vditorEchartsScript").then((function(){a.forEach((function(e){if(!e.parentElement.classList.contains("vditor-wysiwyg__pre")&&!e.parentElement.classList.contains("vditor-ir__marker--pre")){var t=o.mindmapRenderAdapter.getCode(e);if(t)try{if("true"===e.getAttribute("data-processed"))return;echarts.init(e,"dark"===n?"dark":void 0).setOption({series:[{data:[JSON.parse(decodeURIComponent(t))],initialTreeDepth:-1,itemStyle:{borderWidth:0,color:"#4285f4"},label:{backgroundColor:"#f6f8fa",borderColor:"#d1d5da",borderRadius:5,borderWidth:.5,color:"#586069",lineHeight:20,offset:[-5,0],padding:[0,5],position:"insideRight"},lineStyle:{color:"#d1d5da",width:1},roam:!0,symbol:function(e,t){var n;return(null===(n=null==t?void 0:t.data)||void 0===n?void 0:n.children)?"circle":"path://"},type:"tree"}],tooltip:{trigger:"item",triggerOn:"mousemove"}}),e.setAttribute("data-processed","true")}catch(t){e.className="vditor-reset--error",e.innerHTML="mindmap render error: <br>"+t}}}))}))}},198:(e,t,n)=>{"use strict";n.d(t,{k:()=>o});var r=n(615),i=n(323),o=function(e,t,n){var o="",a=[];if(Array.from(e.children).forEach((function(e,t){if((0,r.W)(e)){if(n){var i=e.id.lastIndexOf("_");e.id=e.id.substring(0,-1===i?void 0:i)+"_"+t}a.push(e.id),o+=e.outerHTML.replace("<wbr>","")}})),""===o)return t.innerHTML="","";var l=document.createElement("div");if(n)n.lute.SetToC(!0),"wysiwyg"!==n.currentMode||n.preview.element.contains(e)?"ir"!==n.currentMode||n.preview.element.contains(e)?l.innerHTML=n.lute.HTML2VditorDOM("<p>[ToC]</p>"+o):l.innerHTML=n.lute.SpinVditorIRDOM("<p>[ToC]</p>"+o):l.innerHTML=n.lute.SpinVditorDOM("<p>[ToC]</p>"+o),n.lute.SetToC(n.options.preview.markdown.toc);else{t.classList.add("vditor-outline");var s=Lute.New();s.SetToC(!0),l.innerHTML=s.HTML2VditorDOM("<p>[ToC]</p>"+o)}var d=l.firstElementChild.querySelectorAll("li > span[data-target-id]");return d.forEach((function(e,t){if(e.nextElementSibling&&"UL"===e.nextElementSibling.tagName){var n="<svg class='vditor-outline__action'><use xlink:href='#vditor-icon-down'></use></svg>";document.getElementById("vditorIconScript")||(n='<svg class="vditor-outline__action" viewBox="0 0 32 32"><path d="M3.76 6.12l12.24 12.213 12.24-12.213 3.76 3.76-16 16-16-16 3.76-3.76z"></path></svg>'),e.innerHTML=n+"<span>"+e.innerHTML+"</span>"}else e.innerHTML="<svg></svg><span>"+e.innerHTML+"</span>";e.setAttribute("data-target-id",a[t])})),o=l.firstElementChild.innerHTML,0===d.length?(t.innerHTML="",o):(t.innerHTML=o,n&&(0,i.H)(t,{cdn:n.options.cdn,math:n.options.preview.math}),t.firstElementChild.addEventListener("click",(function(r){for(var i=r.target;i&&!i.isEqualNode(t);){if(i.classList.contains("vditor-outline__action")){i.classList.contains("vditor-outline__action--close")?(i.classList.remove("vditor-outline__action--close"),i.parentElement.nextElementSibling.setAttribute("style","display:block")):(i.classList.add("vditor-outline__action--close"),i.parentElement.nextElementSibling.setAttribute("style","display:none")),r.preventDefault(),r.stopPropagation();break}if(i.getAttribute("data-target-id")){r.preventDefault(),r.stopPropagation();var o=document.getElementById(i.getAttribute("data-target-id"));if(!o)return;if(n)if("auto"===n.options.height){var a=o.offsetTop+n.element.offsetTop;n.options.toolbarConfig.pin||(a+=n.toolbar.element.offsetHeight),window.scrollTo(window.scrollX,a)}else n.element.offsetTop<window.scrollY&&window.scrollTo(window.scrollX,n.element.offsetTop),n.preview.element.contains(e)?e.parentElement.scrollTop=o.offsetTop:e.scrollTop=o.offsetTop;else window.scrollTo(window.scrollX,o.offsetTop);break}i=i.parentElement}})),o)}},583:(e,t,n)=>{"use strict";n.d(t,{B:()=>a});var r=n(260),i=n(228),o=n(46),a=function(e,t){void 0===e&&(e=document),void 0===t&&(t=r.g.CDN);var n=o.plantumlRenderAdapter.getElements(e);0!==n.length&&(0,i.G)(t+"/dist/js/plantuml/plantuml-encoder.min.js","vditorPlantumlScript").then((function(){n.forEach((function(e){if(!e.parentElement.classList.contains("vditor-wysiwyg__pre")&&!e.parentElement.classList.contains("vditor-ir__marker--pre")){var t=o.plantumlRenderAdapter.getCode(e).trim();if(t)try{e.innerHTML='<img src="http://www.plantuml.com/plantuml/svg/~1'+plantumlEncoder.encode(t)+'">'}catch(t){e.className="vditor-reset--error",e.innerHTML="plantuml render error: <br>"+t}}}))}))}},792:(e,t,n)=>{"use strict";n.d(t,{X:()=>r});var r=function(e){var t=Lute.New();return t.PutEmojis(e.emojis),t.SetEmojiSite(e.emojiSite),t.SetHeadingAnchor(e.headingAnchor),t.SetInlineMathAllowDigitAfterOpenMarker(e.inlineMathDigit),t.SetAutoSpace(e.autoSpace),t.SetToC(e.toc),t.SetFootnotes(e.footnotes),t.SetFixTermTypo(e.fixTermTypo),t.SetVditorCodeBlockPreview(e.codeBlockPreview),t.SetVditorMathBlockPreview(e.mathBlockPreview),t.SetSanitize(e.sanitize),t.SetChineseParagraphBeginningSpace(e.paragraphBeginningSpace),t.SetRenderListStyle(e.listStyle),t.SetLinkBase(e.linkBase),t.SetLinkPrefix(e.linkPrefix),t.SetMark(e.mark),e.lazyLoadImage&&t.SetImageLazyLoading(e.lazyLoadImage),t}},264:(e,t,n)=>{"use strict";n.d(t,{E:()=>r});var r=function(e,t,n){void 0===t&&(t="zh_CN"),void 0===n&&(n="classic");var r=e.getBoundingClientRect();document.body.insertAdjacentHTML("beforeend",'<div class="vditor vditor-img'+("dark"===n?" vditor--dark":"")+'">\n    <div class="vditor-img__bar">\n      <span class="vditor-img__btn" data-deg="0">\n        <svg><use xlink:href="#vditor-icon-redo"></use></svg>\n        '+window.VditorI18n.spin+"\n      </span>\n      <span class=\"vditor-img__btn\"  onclick=\"this.parentElement.parentElement.outerHTML = '';document.body.style.overflow = ''\">\n        X &nbsp;"+window.VditorI18n.close+'\n      </span>\n    </div>\n    <div class="vditor-img__img" onclick="this.parentElement.outerHTML = \'\';document.body.style.overflow = \'\'">\n      <img style="width: '+e.width+"px;height:"+e.height+"px;transform: translate3d("+r.left+"px, "+(r.top-36)+'px, 0)" src="'+e.getAttribute("src")+'">\n    </div>\n</div>'),document.body.style.overflow="hidden";var i=document.querySelector(".vditor-img img"),o="translate3d("+Math.max(0,window.innerWidth-e.naturalWidth)/2+"px, "+Math.max(0,window.innerHeight-36-e.naturalHeight)/2+"px, 0)";setTimeout((function(){i.setAttribute("style","transition: transform .3s ease-in-out;transform: "+o),setTimeout((function(){i.parentElement.scrollTo((i.parentElement.scrollWidth-i.parentElement.clientWidth)/2,(i.parentElement.scrollHeight-i.parentElement.clientHeight)/2)}),400)}));var a=document.querySelector(".vditor-img__btn");a.addEventListener("click",(function(){var t=parseInt(a.getAttribute("data-deg"),10)+90;t/90%2==1&&e.naturalWidth>i.parentElement.clientHeight?i.style.transform="translate3d("+Math.max(0,window.innerWidth-e.naturalWidth)/2+"px, "+(e.naturalWidth/2-e.naturalHeight/2)+"px, 0) rotateZ("+t+"deg)":i.style.transform=o+" rotateZ("+t+"deg)",a.setAttribute("data-deg",t.toString()),setTimeout((function(){i.parentElement.scrollTo((i.parentElement.scrollWidth-i.parentElement.clientWidth)/2,(i.parentElement.scrollHeight-i.parentElement.clientHeight)/2)}),400)}))}},968:(e,t,n)=>{"use strict";n.d(t,{Y:()=>o});var r=n(260),i=n(946),o=function(e,t){void 0===t&&(t=r.g.CDN),r.g.CODE_THEME.includes(e)||(e="github");var n=document.getElementById("vditorHljsStyle"),o=t+"/dist/js/highlight.js/styles/"+e+".css";n?n.href!==o&&(n.remove(),(0,i.c)(o,"vditorHljsStyle")):(0,i.c)(o,"vditorHljsStyle")}},958:(e,t,n)=>{"use strict";n.d(t,{Z:()=>i});var r=n(946),i=function(e,t){if(e&&t){var n=document.getElementById("vditorContentTheme"),i=t+"/"+e+".css";n?n.href!==i&&(n.remove(),(0,r.c)(i,"vditorContentTheme")):(0,r.c)(i,"vditorContentTheme")}}},228:(e,t,n)=>{"use strict";n.d(t,{J:()=>r,G:()=>i});var r=function(e,t){if(document.getElementById(t))return!1;var n=new XMLHttpRequest;n.open("GET",e,!1),n.setRequestHeader("Accept","text/javascript, application/javascript, application/ecmascript, application/x-ecmascript, */*; q=0.01"),n.send("");var r=document.createElement("script");r.type="text/javascript",r.text=n.responseText,r.id=t,document.head.appendChild(r)},i=function(e,t){return new Promise((function(n,r){if(document.getElementById(t))return n(),!1;var i=document.createElement("script");i.src=e,i.async=!0,document.head.appendChild(i),i.onload=function(){if(document.getElementById(t))return i.remove(),n(),!1;i.id=t,n()}}))}},946:(e,t,n)=>{"use strict";n.d(t,{c:()=>r});var r=function(e,t){if(!document.getElementById(t)){var n=document.createElement("link");n.id=t,n.rel="stylesheet",n.type="text/css",n.href=e,document.getElementsByTagName("head")[0].appendChild(n)}}},769:(e,t,n)=>{"use strict";n.d(t,{X:()=>r});var r=function(e){return e.replace(/\u00a0/g," ")}},931:(e,t,n)=>{"use strict";n.d(t,{G6:()=>r,vU:()=>i,pK:()=>o,Le:()=>a,yl:()=>l,ns:()=>s,i7:()=>d});var r=function(){return navigator.userAgent.indexOf("Safari")>-1&&-1===navigator.userAgent.indexOf("Chrome")},i=function(){return navigator.userAgent.toLowerCase().indexOf("firefox")>-1},o=function(){try{return"undefined"!=typeof localStorage}catch(e){return!1}},a=function(){return navigator.userAgent.indexOf("iPhone")>-1?"touchstart":"click"},l=function(e){return navigator.platform.toUpperCase().indexOf("MAC")>=0?!(!e.metaKey||e.ctrlKey):!(e.metaKey||!e.ctrlKey)},s=function(e){return/Mac/.test(navigator.platform)||"iPhone"===navigator.platform?e.indexOf("⇧")>-1&&i()&&(e=e.replace(";",":").replace("=","+").replace("-","_")):(e=(e=e.startsWith("⌘")?e.replace("⌘","⌘+"):e.startsWith("⌥")&&"⌘"!==e.substr(1,1)?e.replace("⌥","⌥+"):e.replace("⇧⌘","⌘+⇧+").replace("⌥⌘","⌥+⌘+")).replace("⌘","Ctrl").replace("⇧","Shift").replace("⌥","Alt")).indexOf("Shift")>-1&&(e=e.replace(";",":").replace("=","+").replace("-","_")),e},d=function(){return/Chrome/.test(navigator.userAgent)&&/Google Inc/.test(navigator.vendor)}},713:(e,t,n)=>{"use strict";n.d(t,{JQ:()=>i,E2:()=>o,O9:()=>a,a1:()=>l,F9:()=>s,lG:()=>d,fb:()=>c,DX:()=>u});var r=n(615),i=function(e,t){for(var n=c(e,t),r=!1,i=!1;n&&!n.classList.contains("vditor-reset")&&!i;)(r=c(n.parentElement,t))?n=r:i=!0;return n||!1},o=function(e,t){for(var n=(0,r.S)(e,t),i=!1,o=!1;n&&!n.classList.contains("vditor-reset")&&!o;)(i=(0,r.S)(n.parentElement,t))?n=i:o=!0;return n||!1},a=function(e){var t=o(e,"UL"),n=o(e,"OL"),r=t;return n&&(!t||t&&n.contains(t))&&(r=n),r},l=function(e,t,n){if(!e)return!1;3===e.nodeType&&(e=e.parentElement);for(var r=e,i=!1;r&&!i&&!r.classList.contains("vditor-reset");)r.getAttribute(t)===n?i=!0:r=r.parentElement;return i&&r},s=function(e){if(!e)return!1;3===e.nodeType&&(e=e.parentElement);var t=e,n=!1,r=l(e,"data-block","0");if(r)return r;for(;t&&!n&&!t.classList.contains("vditor-reset");)"H1"===t.tagName||"H2"===t.tagName||"H3"===t.tagName||"H4"===t.tagName||"H5"===t.tagName||"H6"===t.tagName||"P"===t.tagName||"BLOCKQUOTE"===t.tagName||"OL"===t.tagName||"UL"===t.tagName?n=!0:t=t.parentElement;return n&&t},d=function(e,t){if(!e)return!1;3===e.nodeType&&(e=e.parentElement);for(var n=e,r=!1;n&&!r&&!n.classList.contains("vditor-reset");)n.nodeName===t?r=!0:n=n.parentElement;return r&&n},c=function(e,t){if(!e)return!1;3===e.nodeType&&(e=e.parentElement);for(var n=e,r=!1;n&&!r&&!n.classList.contains("vditor-reset");)n.classList.contains(t)?r=!0:n=n.parentElement;return r&&n},u=function(e){for(;e&&e.lastChild;)e=e.lastChild;return e}},615:(e,t,n)=>{"use strict";n.d(t,{S:()=>r,W:()=>i});var r=function(e,t){if(!e)return!1;3===e.nodeType&&(e=e.parentElement);for(var n=e,r=!1;n&&!r&&!n.classList.contains("vditor-reset");)0===n.nodeName.indexOf(t)?r=!0:n=n.parentElement;return r&&n},i=function(e){var t=r(e,"H");return!(!t||2!==t.tagName.length||"HR"===t.tagName)&&t}},224:(e,t,n)=>{"use strict";n.d(t,{T:()=>r});var r=function(){for(var e=[],t=0;t<arguments.length;t++)e[t]=arguments[t];for(var n={},i=function(e){for(var t in e)e.hasOwnProperty(t)&&("[object Object]"===Object.prototype.toString.call(e[t])?n[t]=r(n[t],e[t]):n[t]=e[t])},o=0;o<e.length;o++)i(e[o]);return n}},187:(e,t,n)=>{"use strict";n.d(t,{zh:()=>a,Ny:()=>l,Gb:()=>s,Hc:()=>d,im:()=>c,$j:()=>u,ib:()=>p,oC:()=>m});var r=n(260),i=n(931),o=n(713),a=function(e){var t,n=e[e.currentMode].element;return getSelection().rangeCount>0&&(t=getSelection().getRangeAt(0),n.isEqualNode(t.startContainer)||n.contains(t.startContainer))?t:e[e.currentMode].range?e[e.currentMode].range:(n.focus(),(t=n.ownerDocument.createRange()).setStart(n,0),t.collapse(!0),t)},l=function(e){var t=window.getSelection().getRangeAt(0);if(!e.contains(t.startContainer)&&!(0,o.fb)(t.startContainer,"vditor-panel--none"))return{left:0,top:0};var n,r=e.parentElement.getBoundingClientRect();if(0===t.getClientRects().length)if(3===t.startContainer.nodeType){var i=t.startContainer.parentElement;if(!(i&&i.getClientRects().length>0))return{left:0,top:0};n=i.getClientRects()[0]}else{var a=t.startContainer.children;if(a[t.startOffset]&&a[t.startOffset].getClientRects().length>0)n=a[t.startOffset].getClientRects()[0];else if(t.startContainer.childNodes.length>0){var l=t.cloneRange();t.selectNode(t.startContainer.childNodes[Math.max(0,t.startOffset-1)]),n=t.getClientRects()[0],t.setEnd(l.endContainer,l.endOffset),t.setStart(l.startContainer,l.startOffset)}else n=t.startContainer.getClientRects()[0];if(!n){for(var s=t.startContainer.childNodes[t.startOffset];!s.getClientRects||s.getClientRects&&0===s.getClientRects().length;)s=s.parentElement;n=s.getClientRects()[0]}}else n=t.getClientRects()[0];return{left:n.left-r.left,top:n.top-r.top}},s=function(e,t){if(!t){if(0===getSelection().rangeCount)return!1;t=getSelection().getRangeAt(0)}var n=t.commonAncestorContainer;return e.isEqualNode(n)||e.contains(n)},d=function(e){var t=window.getSelection();t.removeAllRanges(),t.addRange(e)},c=function(e,t,n){var r={end:0,start:0};if(!n){if(0===getSelection().rangeCount)return r;n=window.getSelection().getRangeAt(0)}if(s(t,n)){var i=n.cloneRange();e.childNodes[0]&&e.childNodes[0].childNodes[0]?i.setStart(e.childNodes[0].childNodes[0],0):i.selectNodeContents(e),i.setEnd(n.startContainer,n.startOffset),r.start=i.toString().length,r.end=r.start+n.toString().length}return r},u=function(e,t,n){var r=0,i=0,o=n.childNodes[i],a=!1,l=!1;e=Math.max(0,e),t=Math.max(0,t);var s=n.ownerDocument.createRange();for(s.setStart(o||n,0),s.collapse(!0);!l&&o;){var c=r+o.textContent.length;if(!a&&e>=r&&e<=c&&(0===e?s.setStart(o,0):3===o.childNodes[0].nodeType?s.setStart(o.childNodes[0],e-r):o.nextSibling?s.setStartBefore(o.nextSibling):s.setStartAfter(o),a=!0,e===t)){l=!0;break}a&&t>=r&&t<=c&&(0===t?s.setEnd(o,0):3===o.childNodes[0].nodeType?s.setEnd(o.childNodes[0],t-r):o.nextSibling?s.setEndBefore(o.nextSibling):s.setEndAfter(o),l=!0),r=c,o=n.childNodes[++i]}return!l&&n.childNodes[i-1]&&s.setStartBefore(n.childNodes[i-1]),d(s),s},p=function(e,t){var n=e.querySelector("wbr");if(n){if(n.previousElementSibling)if(n.previousElementSibling.isSameNode(n.previousSibling)){if(n.previousElementSibling.lastChild)return t.setStartBefore(n),t.collapse(!0),d(t),!(0,i.i7)()||"EM"!==n.previousElementSibling.tagName&&"STRONG"!==n.previousElementSibling.tagName&&"S"!==n.previousElementSibling.tagName||(t.insertNode(document.createTextNode(r.g.ZWSP)),t.collapse(!1)),void n.remove();t.setStartAfter(n.previousElementSibling)}else t.setStart(n.previousSibling,n.previousSibling.textContent.length);else n.previousSibling?t.setStart(n.previousSibling,n.previousSibling.textContent.length):n.nextSibling?3===n.nextSibling.nodeType?t.setStart(n.nextSibling,0):t.setStartBefore(n.nextSibling):t.setStart(n.parentElement,0);t.collapse(!0),n.remove(),d(t)}},m=function(e,t){var n=document.createElement("div");n.innerHTML=e;var r=n.querySelectorAll("p");1===r.length&&!r[0].previousSibling&&!r[0].nextSibling&&t[t.currentMode].element.children.length>0&&"P"===n.firstElementChild.tagName&&(e=r[0].innerHTML.trim());var i=document.createElement("div");i.innerHTML=e;var l=a(t);if(""!==l.toString()&&(t[t.currentMode].preventInput=!0,document.execCommand("delete",!1,"")),i.firstElementChild&&"0"===i.firstElementChild.getAttribute("data-block")){i.lastElementChild.insertAdjacentHTML("beforeend","<wbr>");var s=(0,o.F9)(l.startContainer);s?s.insertAdjacentHTML("afterend",i.innerHTML):t[t.currentMode].element.insertAdjacentHTML("beforeend",i.innerHTML),p(t[t.currentMode].element,l)}else{var c=document.createElement("template");c.innerHTML=e,l.insertNode(c.content.cloneNode(!0)),l.collapse(!1),d(l)}}}},t={};function n(r){var i=t[r];if(void 0!==i)return i.exports;var o=t[r]={exports:{}};return e[r](o,o.exports,n),o.exports}n.d=(e,t)=>{for(var r in t)n.o(t,r)&&!n.o(e,r)&&Object.defineProperty(e,r,{enumerable:!0,get:t[r]})},n.o=(e,t)=>Object.prototype.hasOwnProperty.call(e,t),n.r=e=>{"undefined"!=typeof Symbol&&Symbol.toStringTag&&Object.defineProperty(e,Symbol.toStringTag,{value:"Module"}),Object.defineProperty(e,"__esModule",{value:!0})};var r={};return(()=>{"use strict";n.d(r,{default:()=>Gn});n(157);var e,t=n(857),i=n(260),o=n(769),a=function(e){return"sv"===e.currentMode?(0,o.X)((e.sv.element.textContent+"\n").replace(/\n\n$/,"\n")):"wysiwyg"===e.currentMode?e.lute.VditorDOM2Md(e.wysiwyg.element.innerHTML):"ir"===e.currentMode?e.lute.VditorIRDOM2Md(e.ir.element.innerHTML):""},l=n(228),s=function(){function e(){this.element=document.createElement("div"),this.element.className="vditor-devtools",this.element.innerHTML='<div class="vditor-reset--error"></div><div style="height: 100%;"></div>'}return e.prototype.renderEchart=function(e){var t=this;"block"===e.devtools.element.style.display&&(0,l.G)(e.options.cdn+"/dist/js/echarts/echarts.min.js","vditorEchartsScript").then((function(){t.ASTChart||(t.ASTChart=echarts.init(e.devtools.element.lastElementChild));try{t.element.lastElementChild.style.display="block",t.element.firstElementChild.innerHTML="",t.ASTChart.setOption({series:[{data:JSON.parse(e.lute.RenderEChartsJSON(a(e))),initialTreeDepth:-1,label:{align:"left",backgroundColor:"rgba(68, 77, 86, .68)",borderRadius:3,color:"#d1d5da",fontSize:12,lineHeight:12,offset:[9,12],padding:[2,4,2,4],position:"top",verticalAlign:"middle"},lineStyle:{color:"#4285f4",type:"curve",width:1},orient:"vertical",roam:!0,type:"tree"}],toolbox:{bottom:25,emphasis:{iconStyle:{color:"#4285f4"}},feature:{restore:{show:!0},saveAsImage:{show:!0}},right:15,show:!0}}),t.ASTChart.resize()}catch(e){t.element.lastElementChild.style.display="none",t.element.firstElementChild.innerHTML=e}}))},e}(),d=n(931),c=function(e,t){t.forEach((function(t){if(e[t]){var n=e[t].children[0];n&&n.classList.contains("vditor-menu--current")&&n.classList.remove("vditor-menu--current")}}))},u=function(e,t){t.forEach((function(t){if(e[t]){var n=e[t].children[0];n&&!n.classList.contains("vditor-menu--current")&&n.classList.add("vditor-menu--current")}}))},p=function(e,t){t.forEach((function(t){if(e[t]){var n=e[t].children[0];n&&n.classList.contains(i.g.CLASS_MENU_DISABLED)&&n.classList.remove(i.g.CLASS_MENU_DISABLED)}}))},m=function(e,t){t.forEach((function(t){if(e[t]){var n=e[t].children[0];n&&!n.classList.contains(i.g.CLASS_MENU_DISABLED)&&n.classList.add(i.g.CLASS_MENU_DISABLED)}}))},f=function(e,t){t.forEach((function(t){e[t]&&e[t]&&(e[t].style.display="none")}))},h=function(e,t){t.forEach((function(t){e[t]&&e[t]&&(e[t].style.display="block")}))},v=function(e,t,n){t.includes("subToolbar")&&(e.toolbar.element.querySelectorAll(".vditor-hint").forEach((function(e){n&&e.isEqualNode(n)||(e.style.display="none")})),e.toolbar.elements.emoji&&(e.toolbar.elements.emoji.lastElementChild.style.display="none")),t.includes("hint")&&(e.hint.element.style.display="none"),e.wysiwyg.popover&&t.includes("popover")&&(e.wysiwyg.popover.style.display="none")},g=function(e,t,n,r){n.addEventListener((0,d.Le)(),(function(r){r.preventDefault(),r.stopPropagation(),n.classList.contains(i.g.CLASS_MENU_DISABLED)||(e.toolbar.element.querySelectorAll(".vditor-hint--current").forEach((function(e){e.classList.remove("vditor-hint--current")})),"block"===t.style.display?t.style.display="none":(v(e,["subToolbar","hint","popover"],n.parentElement.parentElement),n.classList.contains("vditor-tooltipped")||n.classList.add("vditor-hint--current"),t.style.display="block",e.toolbar.element.getBoundingClientRect().right-n.getBoundingClientRect().right<250?t.classList.add("vditor-panel--left"):t.classList.remove("vditor-panel--left")))}))},y=n(713),b=n(615),w=function(e,t,n,r){r&&console.log(e+" - "+n+": "+t)},E=n(369),k=n(726),S=n(23),C=n(383),L=n(890),T=n(93),M=n(323),A=n(765),_=n(894),x=n(583),H=function(e,t){if(e)if("html-block"!==e.parentElement.getAttribute("data-type")){var n=e.firstElementChild.className.replace("language-","");n&&("abc"===n?(0,E.Q)(e,t.options.cdn):"mermaid"===n?(0,A.i)(e,t.options.cdn,t.options.theme):"flowchart"===n?(0,C.P)(e,t.options.cdn):"echarts"===n?(0,k.p)(e,t.options.cdn,t.options.theme):"mindmap"===n?(0,_.P)(e,t.options.cdn,t.options.theme):"plantuml"===n?(0,x.B)(e,t.options.cdn):"graphviz"===n?(0,L.v)(e,t.options.cdn):"math"===n?(0,M.H)(e,{cdn:t.options.cdn,math:t.options.preview.math}):((0,T.s)(Object.assign({},t.options.preview.hljs),e,t.options.cdn),(0,S.O)(e)),e.setAttribute("data-render","1"))}else e.setAttribute("data-render","1")},N=n(187),D=function(e){if("sv"!==e.currentMode){var t=e[e.currentMode].element,n=e.outline.render(e);""===n&&(n="[ToC]"),t.querySelectorAll('[data-type="toc-block"]').forEach((function(t){t.innerHTML=n,(0,M.H)(t,{cdn:e.options.cdn,math:e.options.preview.math})}))}},O=function(e,t){var n=(0,y.lG)(e.target,"SPAN");if(n&&(0,y.fb)(n,"vditor-toc")){var r=t[t.currentMode].element.querySelector("#"+n.getAttribute("data-target-id"));if(r)if("auto"===t.options.height){var i=r.offsetTop+t.element.offsetTop;t.options.toolbarConfig.pin||(i+=t.toolbar.element.offsetHeight),window.scrollTo(window.scrollX,i)}else t.element.offsetTop<window.scrollY&&window.scrollTo(window.scrollX,t.element.offsetTop),t[t.currentMode].element.scrollTop=r.offsetTop}else;},I=function(e,t,n,r){if(e.previousElementSibling&&e.previousElementSibling.classList.contains("vditor-toc")){if("Backspace"===n.key&&0===(0,N.im)(e,t[t.currentMode].element,r).start)return e.previousElementSibling.remove(),lt(t),!0;if(et(t,n,r,e,e.previousElementSibling))return!0}if(e.nextElementSibling&&e.nextElementSibling.classList.contains("vditor-toc")){if("Delete"===n.key&&(0,N.im)(e,t[t.currentMode].element,r).start>=e.textContent.trimRight().length)return e.nextElementSibling.remove(),lt(t),!0;if($e(t,n,r,e,e.nextElementSibling))return!0}if("Backspace"===n.key||"Delete"===n.key){var i=(0,y.fb)(r.startContainer,"vditor-toc");if(i)return i.remove(),lt(t),!0}},j=function(e,t,n,r){void 0===n&&(n=!1);var o=(0,y.F9)(t.startContainer);if(o&&!n&&"code-block"!==o.getAttribute("data-type")){if(ot(o.innerHTML)&&o.previousElementSibling||at(o.innerHTML))return;for(var a=(0,N.im)(o,e.ir.element,t).start,l=!0,s=a-1;s>o.textContent.substr(0,a).lastIndexOf("\n");s--)if(" "!==o.textContent.charAt(s)&&"\t"!==o.textContent.charAt(s)){l=!1;break}0===a&&(l=!1);var d=!0;for(s=a-1;s<o.textContent.length;s++)if(" "!==o.textContent.charAt(s)&&"\n"!==o.textContent.charAt(s)){d=!1;break}if(l)return;if(d)if(!(0,y.fb)(t.startContainer,"vditor-ir__marker")){var c=t.startContainer.previousSibling;return void(c&&3!==c.nodeType&&c.classList.contains("vditor-ir__node--expand")&&c.classList.remove("vditor-ir__node--expand"))}}if(e.ir.element.querySelectorAll(".vditor-ir__node--expand").forEach((function(e){e.classList.remove("vditor-ir__node--expand")})),o||(o=e.ir.element),!o.querySelector("wbr")){var u=(0,y.fb)(t.startContainer,"vditor-ir__preview");u?u.previousElementSibling.insertAdjacentHTML("beforeend","<wbr>"):t.insertNode(document.createElement("wbr"))}o.querySelectorAll("[style]").forEach((function(e){e.removeAttribute("style")})),"link-ref-defs-block"===o.getAttribute("data-type")&&(o=e.ir.element);var p,m=o.isEqualNode(e.ir.element),f=(0,y.a1)(o,"data-type","footnotes-block"),h="";if(m)h=o.innerHTML;else{var v=(0,b.S)(t.startContainer,"BLOCKQUOTE"),g=(0,y.O9)(t.startContainer);if(g&&(o=g),v&&(!g||g&&!v.contains(g))&&(o=v),f&&(o=f),h=o.outerHTML,"UL"===o.tagName||"OL"===o.tagName){var E=o.previousElementSibling,k=o.nextElementSibling;!E||"UL"!==E.tagName&&"OL"!==E.tagName||(h=E.outerHTML+h,E.remove()),!k||"UL"!==k.tagName&&"OL"!==k.tagName||(h+=k.outerHTML,k.remove()),h=h.replace("<div><wbr><br></div>","<li><p><wbr><br></p></li>")}else o.previousElementSibling&&""!==o.previousElementSibling.textContent.replace(i.g.ZWSP,"")&&r&&"insertParagraph"===r.inputType&&(h=o.previousElementSibling.outerHTML+h,o.previousElementSibling.remove());e.ir.element.querySelectorAll("[data-type='link-ref-defs-block']").forEach((function(e){e&&!o.isEqualNode(e)&&(h+=e.outerHTML,e.remove())})),e.ir.element.querySelectorAll("[data-type='footnotes-block']").forEach((function(e){e&&!o.isEqualNode(e)&&(h+=e.outerHTML,e.remove())}))}if(w("SpinVditorIRDOM",h,"argument",e.options.debugger),h=e.lute.SpinVditorIRDOM(h),w("SpinVditorIRDOM",h,"result",e.options.debugger),m)o.innerHTML=h;else if(o.outerHTML=h,f){var S=(0,y.a1)(e.ir.element.querySelector("wbr"),"data-type","footnotes-def");if(S){var C=S.textContent,L=C.substring(1,C.indexOf("]:")),T=e.ir.element.querySelector('sup[data-type="footnotes-ref"][data-footnotes-label="'+L+'"]');T&&T.setAttribute("aria-label",C.substr(L.length+3).trim().substr(0,24))}}var M,A=e.ir.element.querySelectorAll("[data-type='link-ref-defs-block']");A.forEach((function(e,t){0===t?p=e:(p.insertAdjacentHTML("beforeend",e.innerHTML),e.remove())})),A.length>0&&e.ir.element.insertAdjacentElement("beforeend",A[0]);var _=e.ir.element.querySelectorAll("[data-type='footnotes-block']");_.forEach((function(e,t){0===t?M=e:(M.insertAdjacentHTML("beforeend",e.innerHTML),e.remove())})),_.length>0&&e.ir.element.insertAdjacentElement("beforeend",_[0]),(0,N.ib)(e.ir.element,t),e.ir.element.querySelectorAll(".vditor-ir__preview[data-render='2']").forEach((function(t){H(t,e)})),D(e),Lt(e,{enableAddUndoStack:!0,enableHint:!0,enableInput:!0})},R=function(e,t){if(""===e)return!1;if(-1===e.indexOf("⇧")&&-1===e.indexOf("⌘")&&-1===e.indexOf("⌥"))return!((0,d.yl)(t)||t.altKey||t.shiftKey||t.code!==e);if("⇧Tab"===e)return!((0,d.yl)(t)||t.altKey||!t.shiftKey||"Tab"!==t.code);var n=e.split("");if(e.startsWith("⌥")){var r=3===n.length?n[2]:n[1];return!((3===n.length?!(0,d.yl)(t):(0,d.yl)(t))||!t.altKey||t.shiftKey||t.code!==(/^[0-9]$/.test(r)?"Digit":"Key")+r)}"⌘Enter"===e&&(n=["⌘","Enter"]);var i=n.length>2&&"⇧"===n[0],o=i?n[2]:n[1];return!i||!(0,d.vU)()&&/Mac/.test(navigator.platform)||("-"===o?o="_":"="===o&&(o="+")),!(!(0,d.yl)(t)||t.key.toLowerCase()!==o.toLowerCase()||t.altKey||!(!i&&!t.shiftKey||i&&t.shiftKey))},P=function(e,t){t.ir.element.querySelectorAll(".vditor-ir__node--expand").forEach((function(e){e.classList.remove("vditor-ir__node--expand")}));var n=(0,y.JQ)(e.startContainer,"vditor-ir__node"),r=!e.collapsed&&(0,y.JQ)(e.endContainer,"vditor-ir__node");if(e.collapsed||n&&n===r){n&&(n.classList.add("vditor-ir__node--expand"),n.classList.remove("vditor-ir__node--hidden"),(0,N.Hc)(e));var i=function(e){var t=e.startContainer;if(3===t.nodeType&&t.nodeValue.length!==e.startOffset)return!1;for(var n=t.nextSibling;n&&""===n.textContent;)n=n.nextSibling;if(!n){var r=(0,y.fb)(t,"vditor-ir__marker");if(r&&!r.nextSibling){var i=t.parentElement.parentElement.nextSibling;if(i&&3!==i.nodeType&&i.classList.contains("vditor-ir__node"))return i}return!1}return!(!n||3===n.nodeType||!n.classList.contains("vditor-ir__node")||n.getAttribute("data-block"))&&n}(e);if(i)return i.classList.add("vditor-ir__node--expand"),void i.classList.remove("vditor-ir__node--hidden");var o=function(e){var t=e.startContainer,n=t.previousSibling;return!(3!==t.nodeType||0!==e.startOffset||!n||3===n.nodeType||!n.classList.contains("vditor-ir__node")||n.getAttribute("data-block"))&&n}(e);return o?(o.classList.add("vditor-ir__node--expand"),void o.classList.remove("vditor-ir__node--hidden")):void 0}},B=n(264),q=function(e,t){var n,r=getSelection().getRangeAt(0).cloneRange(),i=r.startContainer;3!==r.startContainer.nodeType&&"DIV"===r.startContainer.tagName&&(i=r.startContainer.childNodes[r.startOffset-1]);var o=(0,y.a1)(i,"data-block","0");if(o&&t&&("deleteContentBackward"===t.inputType||" "===t.data)){for(var a=(0,N.im)(o,e.sv.element,r).start,l=!0,s=a-1;s>o.textContent.substr(0,a).lastIndexOf("\n");s--)if(" "!==o.textContent.charAt(s)&&"\t"!==o.textContent.charAt(s)){l=!1;break}if(0===a&&(l=!1),l)return void De(e);if("deleteContentBackward"===t.inputType){var d=(0,y.a1)(i,"data-type","code-block-open-marker")||(0,y.a1)(i,"data-type","code-block-close-marker");if(d){var c;if("code-block-close-marker"===d.getAttribute("data-type"))if(c=xe(i,"code-block-open-marker"))return c.textContent=d.textContent,void De(e);if("code-block-open-marker"===d.getAttribute("data-type"))if(c=xe(i,"code-block-close-marker",!1))return c.textContent=d.textContent,void De(e)}var u=(0,y.a1)(i,"data-type","math-block-open-marker");if(u){var p=u.nextElementSibling.nextElementSibling;return void(p&&"math-block-close-marker"===p.getAttribute("data-type")&&(p.remove(),De(e)))}o.querySelectorAll('[data-type="code-block-open-marker"]').forEach((function(e){1===e.textContent.length&&e.remove()})),o.querySelectorAll('[data-type="code-block-close-marker"]').forEach((function(e){1===e.textContent.length&&e.remove()}));var m=(0,y.a1)(i,"data-type","heading-marker");if(m&&-1===m.textContent.indexOf("#"))return void De(e)}if((" "===t.data||"deleteContentBackward"===t.inputType)&&((0,y.a1)(i,"data-type","padding")||(0,y.a1)(i,"data-type","li-marker")||(0,y.a1)(i,"data-type","task-marker")||(0,y.a1)(i,"data-type","blockquote-marker")))return void De(e)}if(o&&"$$"===o.textContent.trimRight())De(e);else{o||(o=e.sv.element),"link-ref-defs-block"===(null===(n=o.firstElementChild)||void 0===n?void 0:n.getAttribute("data-type"))&&(o=e.sv.element),(0,y.a1)(i,"data-type","footnotes-link")&&(o=e.sv.element),-1===o.textContent.indexOf(Lute.Caret)&&r.insertNode(document.createTextNode(Lute.Caret)),o.querySelectorAll("[style]").forEach((function(e){e.removeAttribute("style")})),o.querySelectorAll("font").forEach((function(e){e.outerHTML=e.innerHTML}));var f,h=o.textContent,v=o.isEqualNode(e.sv.element);v?h=o.textContent:(o.previousElementSibling&&(h=o.previousElementSibling.textContent+h,o.previousElementSibling.remove()),o.previousElementSibling&&0===h.indexOf("---\n")&&(h=o.previousElementSibling.textContent+h,o.previousElementSibling.remove()),e.sv.element.querySelectorAll("[data-type='link-ref-defs-block']").forEach((function(e,t){0===t&&e&&!o.isEqualNode(e.parentElement)&&(h+="\n"+e.parentElement.textContent,e.parentElement.remove())})),e.sv.element.querySelectorAll("[data-type='footnotes-link']").forEach((function(e,t){0===t&&e&&!o.isEqualNode(e.parentElement)&&(h+="\n"+e.parentElement.textContent,e.parentElement.remove())}))),h=He(h,e),v?o.innerHTML=h:o.outerHTML=h;var g,b=e.sv.element.querySelectorAll("[data-type='link-ref-defs-block']");b.forEach((function(e,t){0===t?f=e.parentElement:(f.lastElementChild.remove(),f.insertAdjacentHTML("beforeend",""+e.parentElement.innerHTML),e.parentElement.remove())})),b.length>0&&e.sv.element.insertAdjacentElement("beforeend",f);var w=e.sv.element.querySelectorAll("[data-type='footnotes-link']");w.forEach((function(e,t){0===t?g=e.parentElement:(g.lastElementChild.remove(),g.insertAdjacentHTML("beforeend",""+e.parentElement.innerHTML),e.parentElement.remove())})),w.length>0&&e.sv.element.insertAdjacentElement("beforeend",g),(0,N.ib)(e.sv.element,r),Te(e),De(e,{enableAddUndoStack:!0,enableHint:!0,enableInput:!0})}},V=n(958),U=function(e){"dark"===e.options.theme?e.element.classList.add("vditor--dark"):e.element.classList.remove("vditor--dark")},W=function(e){var t=window.innerWidth<=i.g.MOBILE_WIDTH?10:35;if("none"!==e.wysiwyg.element.parentElement.style.display){var n=(e.wysiwyg.element.parentElement.clientWidth-e.options.preview.maxWidth)/2;e.wysiwyg.element.style.padding="10px "+Math.max(t,n)+"px"}if("none"!==e.ir.element.parentElement.style.display){n=(e.ir.element.parentElement.clientWidth-e.options.preview.maxWidth)/2;e.ir.element.style.padding="10px "+Math.max(t,n)+"px"}"block"!==e.preview.element.style.display?e.toolbar.element.style.paddingLeft=Math.max(5,parseInt(e[e.currentMode].element.style.paddingLeft||"0",10)+("left"===e.options.outline.position?e.outline.element.offsetWidth:0))+"px":e.toolbar.element.style.paddingLeft=5+("left"===e.options.outline.position?e.outline.element.offsetWidth:0)+"px"},z=function(e){if(e.options.typewriterMode){var t=window.innerHeight;"number"==typeof e.options.height&&(t=e.options.height,"number"==typeof e.options.minHeight&&(t=Math.max(t,e.options.minHeight)),t=Math.min(window.innerHeight,t)),e.element.classList.contains("vditor--fullscreen")&&(t=window.innerHeight),e[e.currentMode].element.style.setProperty("--editor-bottom",(t-e.toolbar.element.offsetHeight)/2+"px")}};function G(){window.removeEventListener("resize",e)}var K,F,Z=function(t){z(t),G(),window.addEventListener("resize",e=function(){W(t),z(t)});var n=(0,d.pK)()&&localStorage.getItem(t.options.cache.id);return t.options.cache.enable&&n||(t.options.value?n=t.options.value:t.originalInnerHTML?n=t.lute.HTML2Md(t.originalInnerHTML):t.options.cache.enable||(n="")),n||""},J=function(e){clearTimeout(e[e.currentMode].hlToolbarTimeoutId),e[e.currentMode].hlToolbarTimeoutId=window.setTimeout((function(){if("false"!==e[e.currentMode].element.getAttribute("contenteditable")&&(0,N.Gb)(e[e.currentMode].element)){c(e.toolbar.elements,i.g.EDIT_TOOLBARS),p(e.toolbar.elements,i.g.EDIT_TOOLBARS);var t=(0,N.zh)(e),n=t.startContainer;3===t.startContainer.nodeType&&(n=t.startContainer.parentElement),n.classList.contains("vditor-reset")&&(n=n.childNodes[t.startOffset]),("sv"===e.currentMode?(0,y.a1)(n,"data-type","heading"):(0,b.W)(n))&&u(e.toolbar.elements,["headings"]),("sv"===e.currentMode?(0,y.a1)(n,"data-type","blockquote"):(0,y.lG)(n,"BLOCKQUOTE"))&&u(e.toolbar.elements,["quote"]),(0,y.a1)(n,"data-type","strong")&&u(e.toolbar.elements,["bold"]),(0,y.a1)(n,"data-type","em")&&u(e.toolbar.elements,["italic"]),(0,y.a1)(n,"data-type","s")&&u(e.toolbar.elements,["strike"]),(0,y.a1)(n,"data-type","a")&&u(e.toolbar.elements,["link"]);var r=(0,y.lG)(n,"LI");r?(r.classList.contains("vditor-task")?u(e.toolbar.elements,["check"]):"OL"===r.parentElement.tagName?u(e.toolbar.elements,["ordered-list"]):"UL"===r.parentElement.tagName&&u(e.toolbar.elements,["list"]),p(e.toolbar.elements,["outdent","indent"])):m(e.toolbar.elements,["outdent","indent"]),(0,y.a1)(n,"data-type","code-block")&&(m(e.toolbar.elements,["headings","bold","italic","strike","line","quote","list","ordered-list","check","code","inline-code","upload","link","table","record"]),u(e.toolbar.elements,["code"])),(0,y.a1)(n,"data-type","code")&&(m(e.toolbar.elements,["headings","bold","italic","strike","line","quote","list","ordered-list","check","code","upload","link","table","record"]),u(e.toolbar.elements,["inline-code"])),(0,y.a1)(n,"data-type","table")&&m(e.toolbar.elements,["headings","list","ordered-list","check","line","quote","code","table"])}}),200)},X=function(e,t){void 0===t&&(t={enableAddUndoStack:!0,enableHint:!1,enableInput:!0}),t.enableHint&&e.hint.render(e),clearTimeout(e.wysiwyg.afterRenderTimeoutId),e.wysiwyg.afterRenderTimeoutId=window.setTimeout((function(){if(!e.wysiwyg.composingLock){var n=a(e);"function"==typeof e.options.input&&t.enableInput&&e.options.input(n),e.options.counter.enable&&e.counter.render(e,n),e.options.cache.enable&&(0,d.pK)()&&(localStorage.setItem(e.options.cache.id,n),e.options.cache.after&&e.options.cache.after(n)),e.devtools&&e.devtools.renderEchart(e),t.enableAddUndoStack&&e.undo.addToUndoStack(e)}}),e.options.undoDelay)},Y=function(e){for(var t="",n=e.nextSibling;n;)3===n.nodeType?t+=n.textContent:t+=n.outerHTML,n=n.nextSibling;return t},Q=function(e){for(var t="",n=e.previousSibling;n;)t=3===n.nodeType?n.textContent+t:n.outerHTML+t,n=n.previousSibling;return t},$=function(e,t){Array.from(e.wysiwyg.element.childNodes).find((function(n){if(3===n.nodeType){var r=document.createElement("p");r.setAttribute("data-block","0"),r.textContent=n.textContent;var i=3===t.startContainer.nodeType?t.startOffset:n.textContent.length;return n.parentNode.insertBefore(r,n),n.remove(),t.setStart(r.firstChild,Math.min(r.firstChild.textContent.length,i)),t.collapse(!0),(0,N.Hc)(t),!0}if(!n.getAttribute("data-block"))return"P"===n.tagName?n.remove():("DIV"===n.tagName?(t.insertNode(document.createElement("wbr")),n.outerHTML='<p data-block="0">'+n.innerHTML+"</p>"):"BR"===n.tagName?n.outerHTML='<p data-block="0">'+n.outerHTML+"<wbr></p>":(t.insertNode(document.createElement("wbr")),n.outerHTML='<p data-block="0">'+n.outerHTML+"</p>"),(0,N.ib)(e.wysiwyg.element,t),t=getSelection().getRangeAt(0)),!0}))},ee=function(e,t){var n=(0,N.zh)(e),r=(0,y.F9)(n.startContainer);r||(r=n.startContainer.childNodes[n.startOffset]),r||0!==e.wysiwyg.element.children.length||(r=e.wysiwyg.element),r&&!r.classList.contains("vditor-wysiwyg__block")&&(n.insertNode(document.createElement("wbr")),"<wbr>"===r.innerHTML.trim()&&(r.innerHTML="<wbr><br>"),"BLOCKQUOTE"===r.tagName||r.classList.contains("vditor-reset")?r.innerHTML="<"+t+' data-block="0">'+r.innerHTML.trim()+"</"+t+">":r.outerHTML="<"+t+' data-block="0">'+r.innerHTML.trim()+"</"+t+">",(0,N.ib)(e.wysiwyg.element,n),D(e))},te=function(e){var t=getSelection().getRangeAt(0),n=(0,y.F9)(t.startContainer);n||(n=t.startContainer.childNodes[t.startOffset]),n&&(t.insertNode(document.createElement("wbr")),n.outerHTML='<p data-block="0">'+n.innerHTML+"</p>",(0,N.ib)(e.wysiwyg.element,t)),e.wysiwyg.popover.style.display="none"},ne=function(e,t,n){void 0===n&&(n=!0);var r=e.previousElementSibling,i=r.ownerDocument.createRange();"CODE"===r.tagName?(r.style.display="inline-block",n?i.setStart(r.firstChild,1):i.selectNodeContents(r)):(r.style.display="block",r.firstChild.firstChild||r.firstChild.appendChild(document.createTextNode("")),i.selectNodeContents(r.firstChild)),n?i.collapse(!0):i.collapse(!1),(0,N.Hc)(i),e.firstElementChild.classList.contains("language-mindmap")||Te(t)},re=function(e,t){if(R("⇧⌘X",t)){var n=e.wysiwyg.popover.querySelector('[data-type="remove"]');if(n)return n.click(),t.preventDefault(),!0}},ie=function(e){clearTimeout(e.wysiwyg.hlToolbarTimeoutId),e.wysiwyg.hlToolbarTimeoutId=window.setTimeout((function(){if("false"!==e.wysiwyg.element.getAttribute("contenteditable")&&(0,N.Gb)(e.wysiwyg.element)){c(e.toolbar.elements,i.g.EDIT_TOOLBARS),p(e.toolbar.elements,i.g.EDIT_TOOLBARS);var t=getSelection().getRangeAt(0),n=t.startContainer;n=3===t.startContainer.nodeType?t.startContainer.parentElement:n.childNodes[t.startOffset>=n.childNodes.length?n.childNodes.length-1:t.startOffset];var r=(0,y.a1)(n,"data-type","footnotes-block");if(r)return e.wysiwyg.popover.innerHTML="",de(r,e),void oe(e,r);var o=(0,y.lG)(n,"LI");o?(o.classList.contains("vditor-task")?u(e.toolbar.elements,["check"]):"OL"===o.parentElement.tagName?u(e.toolbar.elements,["ordered-list"]):"UL"===o.parentElement.tagName&&u(e.toolbar.elements,["list"]),p(e.toolbar.elements,["outdent","indent"])):m(e.toolbar.elements,["outdent","indent"]),(0,y.lG)(n,"BLOCKQUOTE")&&u(e.toolbar.elements,["quote"]),((0,y.lG)(n,"B")||(0,y.lG)(n,"STRONG"))&&u(e.toolbar.elements,["bold"]),((0,y.lG)(n,"I")||(0,y.lG)(n,"EM"))&&u(e.toolbar.elements,["italic"]),((0,y.lG)(n,"STRIKE")||(0,y.lG)(n,"S"))&&u(e.toolbar.elements,["strike"]),e.wysiwyg.element.querySelectorAll(".vditor-comment--focus").forEach((function(e){e.classList.remove("vditor-comment--focus")}));var a=(0,y.fb)(n,"vditor-comment");if(a){var l=a.getAttribute("data-cmtids").split(" ");if(l.length>1&&a.nextSibling.isSameNode(a.nextElementSibling)){var s=a.nextElementSibling.getAttribute("data-cmtids").split(" ");l.find((function(e){if(s.includes(e))return l=[e],!0}))}e.wysiwyg.element.querySelectorAll(".vditor-comment").forEach((function(e){e.getAttribute("data-cmtids").indexOf(l[0])>-1&&e.classList.add("vditor-comment--focus")}))}var f=(0,y.lG)(n,"A");f&&u(e.toolbar.elements,["link"]);var h=(0,y.lG)(n,"TABLE"),v=(0,b.W)(n);(0,y.lG)(n,"CODE")?(0,y.lG)(n,"PRE")?(m(e.toolbar.elements,["headings","bold","italic","strike","line","quote","list","ordered-list","check","code","inline-code","upload","link","table","record"]),u(e.toolbar.elements,["code"])):(m(e.toolbar.elements,["headings","bold","italic","strike","line","quote","list","ordered-list","check","code","upload","link","table","record"]),u(e.toolbar.elements,["inline-code"])):v?(m(e.toolbar.elements,["bold"]),u(e.toolbar.elements,["headings"])):h&&m(e.toolbar.elements,["table"]);var g=(0,y.fb)(n,"vditor-toc");if(g)return e.wysiwyg.popover.innerHTML="",de(g,e),void oe(e,g);var w=(0,b.S)(n,"BLOCKQUOTE");if(w&&(e.wysiwyg.popover.innerHTML="",le(t,w,e),se(t,w,e),de(w,e),oe(e,w)),o&&(e.wysiwyg.popover.innerHTML="",le(t,o,e),se(t,o,e),de(o,e),oe(e,o)),h){e.options.lang,e.options;e.wysiwyg.popover.innerHTML="";var E=function(){var e=h.rows.length,t=h.rows[0].cells.length,n=parseInt(R.value,10)||e,r=parseInt(B.value,10)||t;if(n!==e||t!==r){if(t!==r)for(var i=r-t,o=0;o<h.rows.length;o++)if(i>0)for(var a=0;a<i;a++)0===o?h.rows[o].lastElementChild.insertAdjacentHTML("afterend","<th> </th>"):h.rows[o].lastElementChild.insertAdjacentHTML("afterend","<td> </td>");else for(var l=t-1;l>=r;l--)h.rows[o].cells[l].remove();if(e!==n){var s=n-e;if(s>0){for(var d="<tr>",c=0;c<r;c++)d+="<td> </td>";for(var u=0;u<s;u++)h.querySelector("tbody")?h.querySelector("tbody").insertAdjacentHTML("beforeend",d):h.querySelector("thead").insertAdjacentHTML("afterend",d+"</tr>")}else for(c=e-1;c>=n;c--)h.rows[c].remove(),1===h.rows.length&&h.querySelector("tbody").remove()}}},k=function(n){it(h,n),"right"===n?(T.classList.remove("vditor-icon--current"),M.classList.remove("vditor-icon--current"),A.classList.add("vditor-icon--current")):"center"===n?(T.classList.remove("vditor-icon--current"),A.classList.remove("vditor-icon--current"),M.classList.add("vditor-icon--current")):(M.classList.remove("vditor-icon--current"),A.classList.remove("vditor-icon--current"),T.classList.add("vditor-icon--current")),(0,N.Hc)(t),X(e)},S=(0,y.lG)(n,"TD"),C=(0,y.lG)(n,"TH"),L="left";S?L=S.getAttribute("align")||"left":C&&(L=C.getAttribute("align")||"center");var T=document.createElement("button");T.setAttribute("type","button"),T.setAttribute("aria-label",window.VditorI18n.alignLeft+"<"+(0,d.ns)("⇧⌘L")+">"),T.setAttribute("data-type","left"),T.innerHTML='<svg><use xlink:href="#vditor-icon-align-left"></use></svg>',T.className="vditor-icon vditor-tooltipped vditor-tooltipped__n"+("left"===L?" vditor-icon--current":""),T.onclick=function(){k("left")};var M=document.createElement("button");M.setAttribute("type","button"),M.setAttribute("aria-label",window.VditorI18n.alignCenter+"<"+(0,d.ns)("⇧⌘C")+">"),M.setAttribute("data-type","center"),M.innerHTML='<svg><use xlink:href="#vditor-icon-align-center"></use></svg>',M.className="vditor-icon vditor-tooltipped vditor-tooltipped__n"+("center"===L?" vditor-icon--current":""),M.onclick=function(){k("center")};var A=document.createElement("button");A.setAttribute("type","button"),A.setAttribute("aria-label",window.VditorI18n.alignRight+"<"+(0,d.ns)("⇧⌘R")+">"),A.setAttribute("data-type","right"),A.innerHTML='<svg><use xlink:href="#vditor-icon-align-right"></use></svg>',A.className="vditor-icon vditor-tooltipped vditor-tooltipped__n"+("right"===L?" vditor-icon--current":""),A.onclick=function(){k("right")};var _=document.createElement("button");_.setAttribute("type","button"),_.setAttribute("aria-label",window.VditorI18n.insertRowBelow+"<"+(0,d.ns)("⌘=")+">"),_.setAttribute("data-type","insertRow"),_.innerHTML='<svg><use xlink:href="#vditor-icon-insert-row"></use></svg>',_.className="vditor-icon vditor-tooltipped vditor-tooltipped__n",_.onclick=function(){var n=getSelection().getRangeAt(0).startContainer,r=(0,y.lG)(n,"TD")||(0,y.lG)(n,"TH");r&&ut(e,t,r)};var x=document.createElement("button");x.setAttribute("type","button"),x.setAttribute("aria-label",window.VditorI18n.insertRowAbove+"<"+(0,d.ns)("⇧⌘F")+">"),x.setAttribute("data-type","insertRow"),x.innerHTML='<svg><use xlink:href="#vditor-icon-insert-rowb"></use></svg>',x.className="vditor-icon vditor-tooltipped vditor-tooltipped__n",x.onclick=function(){var n=getSelection().getRangeAt(0).startContainer,r=(0,y.lG)(n,"TD")||(0,y.lG)(n,"TH");r&&pt(e,t,r)};var D=document.createElement("button");D.setAttribute("type","button"),D.setAttribute("aria-label",window.VditorI18n.insertColumnRight+"<"+(0,d.ns)("⇧⌘=")+">"),D.setAttribute("data-type","insertColumn"),D.innerHTML='<svg><use xlink:href="#vditor-icon-insert-column"></use></svg>',D.className="vditor-icon vditor-tooltipped vditor-tooltipped__n",D.onclick=function(){var t=getSelection().getRangeAt(0).startContainer,n=(0,y.lG)(t,"TD")||(0,y.lG)(t,"TH");n&&mt(e,h,n)};var O=document.createElement("button");O.setAttribute("type","button"),O.setAttribute("aria-label",window.VditorI18n.insertColumnLeft+"<"+(0,d.ns)("⇧⌘G")+">"),O.setAttribute("data-type","insertColumn"),O.innerHTML='<svg><use xlink:href="#vditor-icon-insert-columnb"></use></svg>',O.className="vditor-icon vditor-tooltipped vditor-tooltipped__n",O.onclick=function(){var t=getSelection().getRangeAt(0).startContainer,n=(0,y.lG)(t,"TD")||(0,y.lG)(t,"TH");n&&mt(e,h,n,"beforebegin")};var I=document.createElement("button");I.setAttribute("type","button"),I.setAttribute("aria-label",window.VditorI18n["delete-row"]+"<"+(0,d.ns)("⌘-")+">"),I.setAttribute("data-type","deleteRow"),I.innerHTML='<svg><use xlink:href="#vditor-icon-delete-row"></use></svg>',I.className="vditor-icon vditor-tooltipped vditor-tooltipped__n",I.onclick=function(){var n=getSelection().getRangeAt(0).startContainer,r=(0,y.lG)(n,"TD")||(0,y.lG)(n,"TH");r&&ft(e,t,r)};var j=document.createElement("button");j.setAttribute("type","button"),j.setAttribute("aria-label",window.VditorI18n["delete-column"]+"<"+(0,d.ns)("⇧⌘-")+">"),j.setAttribute("data-type","deleteColumn"),j.innerHTML='<svg><use xlink:href="#vditor-icon-delete-column"></use></svg>',j.className="vditor-icon vditor-tooltipped vditor-tooltipped__n",j.onclick=function(){var n=getSelection().getRangeAt(0).startContainer,r=(0,y.lG)(n,"TD")||(0,y.lG)(n,"TH");r&&ht(e,t,h,r)},(F=document.createElement("span")).setAttribute("aria-label",window.VditorI18n.row),F.className="vditor-tooltipped vditor-tooltipped__n";var R=document.createElement("input");F.appendChild(R),R.type="number",R.min="1",R.className="vditor-input",R.style.width="42px",R.style.textAlign="center",R.setAttribute("placeholder",window.VditorI18n.row),R.value=h.rows.length.toString(),R.oninput=function(){E()},R.onkeydown=function(t){if(!t.isComposing)return"Tab"===t.key?(B.focus(),B.select(),void t.preventDefault()):void re(e,t)};var P=document.createElement("span");P.setAttribute("aria-label",window.VditorI18n.column),P.className="vditor-tooltipped vditor-tooltipped__n";var B=document.createElement("input");P.appendChild(B),B.type="number",B.min="1",B.className="vditor-input",B.style.width="42px",B.style.textAlign="center",B.setAttribute("placeholder",window.VditorI18n.column),B.value=h.rows[0].cells.length.toString(),B.oninput=function(){E()},B.onkeydown=function(t){if(!t.isComposing)return"Tab"===t.key?(R.focus(),R.select(),void t.preventDefault()):void re(e,t)},le(t,h,e),se(t,h,e),de(h,e),e.wysiwyg.popover.insertAdjacentElement("beforeend",T),e.wysiwyg.popover.insertAdjacentElement("beforeend",M),e.wysiwyg.popover.insertAdjacentElement("beforeend",A),e.wysiwyg.popover.insertAdjacentElement("beforeend",x),e.wysiwyg.popover.insertAdjacentElement("beforeend",_),e.wysiwyg.popover.insertAdjacentElement("beforeend",O),e.wysiwyg.popover.insertAdjacentElement("beforeend",D),e.wysiwyg.popover.insertAdjacentElement("beforeend",I),e.wysiwyg.popover.insertAdjacentElement("beforeend",j),e.wysiwyg.popover.insertAdjacentElement("beforeend",F),e.wysiwyg.popover.insertAdjacentHTML("beforeend"," x "),e.wysiwyg.popover.insertAdjacentElement("beforeend",P),oe(e,h)}var q=(0,y.a1)(n,"data-type","link-ref");q&&ae(e,q);var V=(0,y.a1)(n,"data-type","footnotes-ref");if(V){e.options.lang,e.options;e.wysiwyg.popover.innerHTML="",(F=document.createElement("span")).setAttribute("aria-label",window.VditorI18n.footnoteRef+"<"+(0,d.ns)("⌥Enter")+">"),F.className="vditor-tooltipped vditor-tooltipped__n";var U=document.createElement("input");F.appendChild(U),U.className="vditor-input",U.setAttribute("placeholder",window.VditorI18n.footnoteRef+"<"+(0,d.ns)("⌥Enter")+">"),U.style.width="120px",U.value=V.getAttribute("data-footnotes-label"),U.oninput=function(){""!==U.value.trim()&&V.setAttribute("data-footnotes-label",U.value)},U.onkeydown=function(n){if(!n.isComposing)return(0,d.yl)(n)||n.shiftKey||!n.altKey||"Enter"!==n.key?void re(e,n):(t.selectNodeContents(V),t.collapse(!1),(0,N.Hc)(t),void n.preventDefault())},de(V,e),e.wysiwyg.popover.insertAdjacentElement("beforeend",F),oe(e,V)}var W=(0,y.fb)(n,"vditor-wysiwyg__block");if(W&&W.getAttribute("data-type").indexOf("block")>-1){e.options.lang,e.options;if(e.wysiwyg.popover.innerHTML="",le(t,W,e),se(t,W,e),de(W,e),"code-block"===W.getAttribute("data-type")){var z=document.createElement("span");z.setAttribute("aria-label",window.VditorI18n.language+"<"+(0,d.ns)("⌥Enter")+">"),z.className="vditor-tooltipped vditor-tooltipped__n";var G=document.createElement("input");z.appendChild(G);var K=W.firstElementChild.firstElementChild;G.className="vditor-input",G.setAttribute("placeholder",window.VditorI18n.language+"<"+(0,d.ns)("⌥Enter")+">"),G.value=K.className.indexOf("language-")>-1?K.className.split("-")[1].split(" ")[0]:"",G.oninput=function(n){""!==G.value.trim()?K.className="language-"+G.value:(K.className="",e.hint.recentLanguage=""),W.lastElementChild.classList.contains("vditor-wysiwyg__preview")&&(W.lastElementChild.innerHTML=W.firstElementChild.innerHTML,H(W.lastElementChild,e)),X(e),1===n.detail&&(t.setStart(K.firstChild,0),t.collapse(!0),(0,N.Hc)(t))},G.onkeydown=function(n){if(!n.isComposing&&!re(e,n)){if("Escape"===n.key&&"block"===e.hint.element.style.display)return e.hint.element.style.display="none",void n.preventDefault();e.hint.select(n,e),(0,d.yl)(n)||n.shiftKey||"Enter"!==n.key||(t.setStart(K.firstChild,0),t.collapse(!0),(0,N.Hc)(t),n.preventDefault(),n.stopPropagation())}},G.onkeyup=function(t){if(!t.isComposing&&"Enter"!==t.key&&"ArrowUp"!==t.key&&"Escape"!==t.key&&"ArrowDown"!==t.key){var n=[],r=G.value.substring(0,G.selectionStart);i.g.CODE_LANGUAGES.forEach((function(e){e.indexOf(r.toLowerCase())>-1&&n.push({html:e,value:e})})),e.hint.genHTML(n,r,e),t.preventDefault()}},e.wysiwyg.popover.insertAdjacentElement("beforeend",z)}oe(e,W)}else W||e.wysiwyg.element.querySelectorAll(".vditor-wysiwyg__preview").forEach((function(e){e.previousElementSibling.style.display="none"})),W=void 0;if(v){var F;e.wysiwyg.popover.innerHTML="",(F=document.createElement("span")).setAttribute("aria-label","ID<"+(0,d.ns)("⌥Enter")+">"),F.className="vditor-tooltipped vditor-tooltipped__n";var Z=document.createElement("input");F.appendChild(Z),Z.className="vditor-input",Z.setAttribute("placeholder","ID<"+(0,d.ns)("⌥Enter")+">"),Z.style.width="120px",Z.value=v.getAttribute("data-id")||"",Z.oninput=function(){v.setAttribute("data-id",Z.value)},Z.onkeydown=function(n){if(!n.isComposing)return(0,d.yl)(n)||n.shiftKey||!n.altKey||"Enter"!==n.key?void re(e,n):(t.selectNodeContents(v),t.collapse(!1),(0,N.Hc)(t),void n.preventDefault())},le(t,v,e),se(t,v,e),de(v,e),e.wysiwyg.popover.insertAdjacentElement("beforeend",F),oe(e,v)}if(f&&ue(e,f),!(w||o||h||W||f||q||V||v||g)){var J=(0,y.a1)(n,"data-block","0");J&&J.parentElement.isEqualNode(e.wysiwyg.element)?(e.wysiwyg.popover.innerHTML="",le(t,J,e),se(t,J,e),de(J,e),oe(e,J)):e.wysiwyg.popover.style.display="none"}e.wysiwyg.element.querySelectorAll('span[data-type="backslash"] > span').forEach((function(e){e.style.display="none"}));var Y=(0,y.a1)(t.startContainer,"data-type","backslash");Y&&(Y.querySelector("span").style.display="inline")}}),200)},oe=function(e,t){var n=t,r=(0,y.lG)(t,"TABLE");r&&(n=r),e.wysiwyg.popover.style.left="0",e.wysiwyg.popover.style.display="block",e.wysiwyg.popover.style.top=Math.max(-8,n.offsetTop-21-e.wysiwyg.element.scrollTop)+"px",e.wysiwyg.popover.style.left=Math.min(n.offsetLeft,e.wysiwyg.element.clientWidth-e.wysiwyg.popover.clientWidth)+"px",e.wysiwyg.popover.setAttribute("data-top",(n.offsetTop-21).toString())},ae=function(e,t){e.wysiwyg.popover.innerHTML="";var n=function(){""!==i.value.trim()&&("IMG"===t.tagName?t.setAttribute("alt",i.value):t.textContent=i.value),""!==a.value.trim()&&t.setAttribute("data-link-label",a.value)},r=document.createElement("span");r.setAttribute("aria-label",window.VditorI18n.textIsNotEmpty),r.className="vditor-tooltipped vditor-tooltipped__n";var i=document.createElement("input");r.appendChild(i),i.className="vditor-input",i.setAttribute("placeholder",window.VditorI18n.textIsNotEmpty),i.style.width="120px",i.value=t.getAttribute("alt")||t.textContent,i.oninput=function(){n()},i.onkeydown=function(n){re(e,n)||ce(e,t,n,a)};var o=document.createElement("span");o.setAttribute("aria-label",window.VditorI18n.linkRef),o.className="vditor-tooltipped vditor-tooltipped__n";var a=document.createElement("input");o.appendChild(a),a.className="vditor-input",a.setAttribute("placeholder",window.VditorI18n.linkRef),a.value=t.getAttribute("data-link-label"),a.oninput=function(){n()},a.onkeydown=function(n){re(e,n)||ce(e,t,n,i)},de(t,e),e.wysiwyg.popover.insertAdjacentElement("beforeend",r),e.wysiwyg.popover.insertAdjacentElement("beforeend",o),oe(e,t)},le=function(e,t,n){var r=t.previousElementSibling;if(r&&(t.parentElement.isEqualNode(n.wysiwyg.element)||"LI"===t.tagName)){var i=document.createElement("button");i.setAttribute("type","button"),i.setAttribute("data-type","up"),i.setAttribute("aria-label",window.VditorI18n.up+"<"+(0,d.ns)("⇧⌘U")+">"),i.innerHTML='<svg><use xlink:href="#vditor-icon-up"></use></svg>',i.className="vditor-icon vditor-tooltipped vditor-tooltipped__n",i.onclick=function(){e.insertNode(document.createElement("wbr")),r.insertAdjacentElement("beforebegin",t),(0,N.ib)(n.wysiwyg.element,e),X(n),ie(n),Te(n)},n.wysiwyg.popover.insertAdjacentElement("beforeend",i)}},se=function(e,t,n){var r=t.nextElementSibling;if(r&&(t.parentElement.isEqualNode(n.wysiwyg.element)||"LI"===t.tagName)){var i=document.createElement("button");i.setAttribute("type","button"),i.setAttribute("data-type","down"),i.setAttribute("aria-label",window.VditorI18n.down+"<"+(0,d.ns)("⇧⌘D")+">"),i.innerHTML='<svg><use xlink:href="#vditor-icon-down"></use></svg>',i.className="vditor-icon vditor-tooltipped vditor-tooltipped__n",i.onclick=function(){e.insertNode(document.createElement("wbr")),r.insertAdjacentElement("afterend",t),(0,N.ib)(n.wysiwyg.element,e),X(n),ie(n),Te(n)},n.wysiwyg.popover.insertAdjacentElement("beforeend",i)}},de=function(e,t){var n=document.createElement("button");n.setAttribute("type","button"),n.setAttribute("data-type","remove"),n.setAttribute("aria-label",window.VditorI18n.remove+"<"+(0,d.ns)("⇧⌘X")+">"),n.innerHTML='<svg><use xlink:href="#vditor-icon-trashcan"></use></svg>',n.className="vditor-icon vditor-tooltipped vditor-tooltipped__n",n.onclick=function(){var n=(0,N.zh)(t);n.setStartAfter(e),(0,N.Hc)(n),e.remove(),X(t),ie(t)},t.wysiwyg.popover.insertAdjacentElement("beforeend",n)},ce=function(e,t,n,r){if(!n.isComposing){if("Tab"===n.key)return r.focus(),r.select(),void n.preventDefault();if(!(0,d.yl)(n)&&!n.shiftKey&&n.altKey&&"Enter"===n.key){var o=(0,N.zh)(e);t.insertAdjacentHTML("afterend",i.g.ZWSP),o.setStartAfter(t.nextSibling),o.collapse(!0),(0,N.Hc)(o),n.preventDefault()}}},ue=function(e,t){e.wysiwyg.popover.innerHTML="";var n=function(){""!==i.value.trim()&&(t.innerHTML=i.value),t.setAttribute("href",a.value),t.setAttribute("title",s.value),X(e)};t.querySelectorAll("[data-marker]").forEach((function(e){e.removeAttribute("data-marker")}));var r=document.createElement("span");r.setAttribute("aria-label",window.VditorI18n.textIsNotEmpty),r.className="vditor-tooltipped vditor-tooltipped__n";var i=document.createElement("input");r.appendChild(i),i.className="vditor-input",i.setAttribute("placeholder",window.VditorI18n.textIsNotEmpty),i.style.width="120px",i.value=t.innerHTML||"",i.oninput=function(){n()},i.onkeydown=function(n){re(e,n)||ce(e,t,n,a)};var o=document.createElement("span");o.setAttribute("aria-label",window.VditorI18n.link),o.className="vditor-tooltipped vditor-tooltipped__n";var a=document.createElement("input");o.appendChild(a),a.className="vditor-input",a.setAttribute("placeholder",window.VditorI18n.link),a.value=t.getAttribute("href")||"",a.oninput=function(){n()},a.onkeydown=function(n){re(e,n)||ce(e,t,n,s)};var l=document.createElement("span");l.setAttribute("aria-label",window.VditorI18n.tooltipText),l.className="vditor-tooltipped vditor-tooltipped__n";var s=document.createElement("input");l.appendChild(s),s.className="vditor-input",s.setAttribute("placeholder",window.VditorI18n.tooltipText),s.style.width="60px",s.value=t.getAttribute("title")||"",s.oninput=function(){n()},s.onkeydown=function(n){re(e,n)||ce(e,t,n,i)},de(t,e),e.wysiwyg.popover.insertAdjacentElement("beforeend",r),e.wysiwyg.popover.insertAdjacentElement("beforeend",o),e.wysiwyg.popover.insertAdjacentElement("beforeend",l),oe(e,t)},pe=function(e){"wysiwyg"===e.currentMode?ie(e):"ir"===e.currentMode&&J(e)},me=function(e,t,n){void 0===n&&(n={enableAddUndoStack:!0,enableHint:!1,enableInput:!0});var r=e.wysiwyg.element;r.innerHTML=e.lute.Md2VditorDOM(t),r.querySelectorAll(".vditor-wysiwyg__preview[data-render='2']").forEach((function(t){H(t,e),t.previousElementSibling.setAttribute("style","display:none")})),X(e,n)},fe=function(e,t,n){for(var r=e.startContainer.parentElement,o=!1,a="",l="",s=function(e){var t=Q(e.startContainer),n=Y(e.startContainer),r=e.startContainer.textContent,o=e.startOffset,a="",l="";return(""!==r.substr(0,o)&&r.substr(0,o)!==i.g.ZWSP||t)&&(a=""+t+r.substr(0,o)),(""!==r.substr(o)&&r.substr(o)!==i.g.ZWSP||n)&&(l=""+r.substr(o)+n),{afterHTML:l,beforeHTML:a}}(e),d=s.beforeHTML,c=s.afterHTML;r&&!o;){var u=r.tagName;if("STRIKE"===u&&(u="S"),"I"===u&&(u="EM"),"B"===u&&(u="STRONG"),"S"===u||"STRONG"===u||"EM"===u){var p="",m="",f="";"0"!==r.parentElement.getAttribute("data-block")&&(m=Q(r),f=Y(r)),(d||m)&&(d=p=m+"<"+u+">"+d+"</"+u+">"),("bold"===n&&"STRONG"===u||"italic"===n&&"EM"===u||"strikeThrough"===n&&"S"===u)&&(p+=""+a+i.g.ZWSP+"<wbr>"+l,o=!0),(c||f)&&(p+=c="<"+u+">"+c+"</"+u+">"+f),"0"!==r.parentElement.getAttribute("data-block")?(r=r.parentElement).innerHTML=p:(r.outerHTML=p,r=r.parentElement),a="<"+u+">"+a,l="</"+u+">"+l}else o=!0}(0,N.ib)(t.wysiwyg.element,e)},he=function(e,t){var n,r=this;this.element=document.createElement("div"),t.className&&(n=this.element.classList).add.apply(n,t.className.split(" "));var o=t.hotkey?" <"+(0,d.ns)(t.hotkey)+">":"";2===t.level&&(o=t.hotkey?" &lt;"+(0,d.ns)(t.hotkey)+"&gt;":"");var a=t.tip?t.tip+o:""+window.VditorI18n[t.name]+o,l="upload"===t.name?"div":"button";if(2===t.level)this.element.innerHTML="<"+l+' data-type="'+t.name+'">'+a+"</"+l+">";else{this.element.classList.add("vditor-toolbar__item");var s=document.createElement(l);s.setAttribute("data-type",t.name),s.className="vditor-tooltipped vditor-tooltipped__"+t.tipPosition,s.setAttribute("aria-label",a),s.innerHTML=t.icon,this.element.appendChild(s)}t.prefix&&this.element.children[0].addEventListener((0,d.Le)(),(function(n){n.preventDefault(),r.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)||("wysiwyg"===e.currentMode?function(e,t,n){if(!(e.wysiwyg.composingLock&&n instanceof CustomEvent)){var r=!0,o=!0;e.wysiwyg.element.querySelector("wbr")&&e.wysiwyg.element.querySelector("wbr").remove();var a=(0,N.zh)(e),l=t.getAttribute("data-type");if(t.classList.contains("vditor-menu--current"))if("strike"===l&&(l="strikeThrough"),"quote"===l){var s=(0,y.lG)(a.startContainer,"BLOCKQUOTE");s||(s=a.startContainer.childNodes[a.startOffset]),s&&(r=!1,t.classList.remove("vditor-menu--current"),a.insertNode(document.createElement("wbr")),s.outerHTML=""===s.innerHTML.trim()?'<p data-block="0">'+s.innerHTML+"</p>":s.innerHTML,(0,N.ib)(e.wysiwyg.element,a))}else if("inline-code"===l){var d=(0,y.lG)(a.startContainer,"CODE");d||(d=a.startContainer.childNodes[a.startOffset]),d&&(d.outerHTML=d.innerHTML.replace(i.g.ZWSP,"")+"<wbr>",(0,N.ib)(e.wysiwyg.element,a))}else"link"===l?a.collapsed?(a.selectNode(a.startContainer.parentElement),document.execCommand("unlink",!1,"")):document.execCommand("unlink",!1,""):"check"===l||"list"===l||"ordered-list"===l?(tt(e,a,l),(0,N.ib)(e.wysiwyg.element,a),r=!1,t.classList.remove("vditor-menu--current")):(r=!1,t.classList.remove("vditor-menu--current"),""===a.toString()?fe(a,e,l):document.execCommand(l,!1,""));else{0===e.wysiwyg.element.childNodes.length&&(e.wysiwyg.element.innerHTML='<p data-block="0"><wbr></p>',(0,N.ib)(e.wysiwyg.element,a));var u=(0,y.F9)(a.startContainer);if("quote"===l){if(u||(u=a.startContainer.childNodes[a.startOffset]),u){r=!1,t.classList.add("vditor-menu--current"),a.insertNode(document.createElement("wbr"));var p=(0,y.lG)(a.startContainer,"LI");p&&u.contains(p)?p.innerHTML='<blockquote data-block="0">'+p.innerHTML+"</blockquote>":u.outerHTML='<blockquote data-block="0">'+u.outerHTML+"</blockquote>",(0,N.ib)(e.wysiwyg.element,a)}}else if("check"===l||"list"===l||"ordered-list"===l)tt(e,a,l,!1),(0,N.ib)(e.wysiwyg.element,a),r=!1,c(e.toolbar.elements,["check","list","ordered-list"]),t.classList.add("vditor-menu--current");else if("inline-code"===l){if(""===a.toString())(m=document.createElement("code")).textContent=i.g.ZWSP,a.insertNode(m),a.setStart(m.firstChild,1),a.collapse(!0),(0,N.Hc)(a);else if(3===a.startContainer.nodeType){var m=document.createElement("code");a.surroundContents(m),a.insertNode(m),(0,N.Hc)(a)}t.classList.add("vditor-menu--current")}else if("code"===l)(m=document.createElement("div")).className="vditor-wysiwyg__block",m.setAttribute("data-type","code-block"),m.setAttribute("data-block","0"),m.setAttribute("data-marker","```"),""===a.toString()?m.innerHTML="<pre><code><wbr>\n</code></pre>":(m.innerHTML="<pre><code>"+a.toString()+"<wbr></code></pre>",a.deleteContents()),a.insertNode(m),u&&(u.outerHTML=e.lute.SpinVditorDOM(u.outerHTML)),(0,N.ib)(e.wysiwyg.element,a),e.wysiwyg.element.querySelectorAll(".vditor-wysiwyg__preview[data-render='2']").forEach((function(t){H(t,e)})),t.classList.add("vditor-menu--disabled");else if("link"===l){if(""===a.toString()){var f=document.createElement("a");f.innerText=i.g.ZWSP,a.insertNode(f),a.setStart(f.firstChild,1),a.collapse(!0),ue(e,f);var h=e.wysiwyg.popover.querySelector("input");h.value="",h.focus(),o=!1}else{(m=document.createElement("a")).setAttribute("href",""),m.innerHTML=a.toString(),a.surroundContents(m),a.insertNode(m),(0,N.Hc)(a),ue(e,m);var v=e.wysiwyg.popover.querySelectorAll("input");v[0].value=m.innerText,v[1].focus()}r=!1,t.classList.add("vditor-menu--current")}else if("table"===l){var g='<table data-block="0"><thead><tr><th>col1<wbr></th><th>col2</th><th>col3</th></tr></thead><tbody><tr><td> </td><td> </td><td> </td></tr><tr><td> </td><td> </td><td> </td></tr></tbody></table>';if(""===a.toString().trim())u&&""===u.innerHTML.trim().replace(i.g.ZWSP,"")?u.outerHTML=g:document.execCommand("insertHTML",!1,g),a.selectNode(e.wysiwyg.element.querySelector("wbr").previousSibling),e.wysiwyg.element.querySelector("wbr").remove(),(0,N.Hc)(a);else{g='<table data-block="0"><thead><tr>';var b=a.toString().split("\n"),w=b[0].split(",").length>b[0].split("\t").length?",":"\t";b.forEach((function(e,t){0===t?(e.split(w).forEach((function(e,t){g+=0===t?"<th>"+e+"<wbr></th>":"<th>"+e+"</th>"})),g+="</tr></thead>"):(g+=1===t?"<tbody><tr>":"<tr>",e.split(w).forEach((function(e){g+="<td>"+e+"</td>"})),g+="</tr>")})),g+="</tbody></table>",document.execCommand("insertHTML",!1,g),(0,N.ib)(e.wysiwyg.element,a)}r=!1,t.classList.add("vditor-menu--disabled")}else if("line"===l){if(u){var E='<hr data-block="0"><p data-block="0"><wbr>\n</p>';""===u.innerHTML.trim()?u.outerHTML=E:u.insertAdjacentHTML("afterend",E),(0,N.ib)(e.wysiwyg.element,a)}}else if(r=!1,t.classList.add("vditor-menu--current"),"strike"===l&&(l="strikeThrough"),""!==a.toString()||"bold"!==l&&"italic"!==l&&"strikeThrough"!==l)document.execCommand(l,!1,"");else{var k="strong";"italic"===l?k="em":"strikeThrough"===l&&(k="s"),(m=document.createElement(k)).textContent=i.g.ZWSP,a.insertNode(m),m.previousSibling&&m.previousSibling.textContent===i.g.ZWSP&&(m.previousSibling.textContent=""),a.setStart(m.firstChild,1),a.collapse(!0),(0,N.Hc)(a)}}r&&ie(e),o&&X(e)}}(e,r.element.children[0],n):"ir"===e.currentMode?At(e,r.element.children[0],t.prefix||"",t.suffix||""):Ie(e,r.element.children[0],t.prefix||"",t.suffix||""))}))},ve=(K=function(e,t){return K=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},K(e,t)},function(e,t){function n(){this.constructor=e}K(e,t),e.prototype=null===t?Object.create(t):(n.prototype=t.prototype,new n)}),ge=function(e,t,n){var r;if("string"!=typeof n?(v(e,["subToolbar","hint"]),n.preventDefault(),r=a(e)):r=n,e.currentMode!==t||"string"==typeof n){if(e.devtools&&e.devtools.renderEchart(e),"both"===e.options.preview.mode&&"sv"===t?e.preview.element.style.display="block":e.preview.element.style.display="none",p(e.toolbar.elements,i.g.EDIT_TOOLBARS),c(e.toolbar.elements,i.g.EDIT_TOOLBARS),m(e.toolbar.elements,["outdent","indent"]),"ir"===t)f(e.toolbar.elements,["both"]),h(e.toolbar.elements,["outdent","indent","outline","insert-before","insert-after"]),e.sv.element.style.display="none",e.wysiwyg.element.parentElement.style.display="none",e.ir.element.parentElement.style.display="block",e.lute.SetVditorIR(!0),e.lute.SetVditorWYSIWYG(!1),e.lute.SetVditorSV(!1),e.currentMode="ir",e.ir.element.innerHTML=e.lute.Md2VditorIRDOM(r),Lt(e,{enableAddUndoStack:!0,enableHint:!1,enableInput:!1}),W(e),e.ir.element.querySelectorAll(".vditor-ir__preview[data-render='2']").forEach((function(t){H(t,e)})),e.ir.element.querySelectorAll(".vditor-toc").forEach((function(t){(0,M.H)(t,{cdn:e.options.cdn,math:e.options.preview.math})}));else if("wysiwyg"===t)f(e.toolbar.elements,["both"]),h(e.toolbar.elements,["outdent","indent","outline","insert-before","insert-after"]),e.sv.element.style.display="none",e.wysiwyg.element.parentElement.style.display="block",e.ir.element.parentElement.style.display="none",e.lute.SetVditorIR(!1),e.lute.SetVditorWYSIWYG(!0),e.lute.SetVditorSV(!1),e.currentMode="wysiwyg",W(e),me(e,r,{enableAddUndoStack:!0,enableHint:!1,enableInput:!1}),e.wysiwyg.element.querySelectorAll(".vditor-toc").forEach((function(t){(0,M.H)(t,{cdn:e.options.cdn,math:e.options.preview.math})})),e.wysiwyg.popover.style.display="none";else if("sv"===t){h(e.toolbar.elements,["both"]),f(e.toolbar.elements,["outdent","indent","outline","insert-before","insert-after"]),e.wysiwyg.element.parentElement.style.display="none",e.ir.element.parentElement.style.display="none",("both"===e.options.preview.mode||"editor"===e.options.preview.mode)&&(e.sv.element.style.display="block"),e.lute.SetVditorIR(!1),e.lute.SetVditorWYSIWYG(!1),e.lute.SetVditorSV(!0),e.currentMode="sv";var o=He(r,e);"<div data-block='0'></div>"===o&&(o=""),e.sv.element.innerHTML=o,De(e,{enableAddUndoStack:!0,enableHint:!1,enableInput:!1}),W(e)}e.undo.resetIcon(e),"string"!=typeof n&&(e[e.currentMode].element.focus(),pe(e)),D(e),z(e),e.toolbar.elements["edit-mode"]&&(e.toolbar.elements["edit-mode"].querySelectorAll("button").forEach((function(e){e.classList.remove("vditor-menu--current")})),e.toolbar.elements["edit-mode"].querySelector('button[data-mode="'+e.currentMode+'"]').classList.add("vditor-menu--current")),e.outline.toggle(e,"sv"!==e.currentMode&&e.options.outline.enable)}},ye=function(e){function t(t,n){var r=e.call(this,t,n)||this,i=document.createElement("div");return i.className="vditor-hint"+(2===n.level?"":" vditor-panel--arrow"),i.innerHTML='<button data-mode="wysiwyg">'+window.VditorI18n.wysiwyg+" &lt;"+(0,d.ns)("⌥⌘7")+'></button>\n<button data-mode="ir">'+window.VditorI18n.instantRendering+" &lt;"+(0,d.ns)("⌥⌘8")+'></button>\n<button data-mode="sv">'+window.VditorI18n.splitView+" &lt;"+(0,d.ns)("⌥⌘9")+"></button>",r.element.appendChild(i),r._bindEvent(t,i,n),r}return ve(t,e),t.prototype._bindEvent=function(e,t,n){var r=this.element.children[0];g(e,t,r,n.level),t.children.item(0).addEventListener((0,d.Le)(),(function(t){ge(e,"wysiwyg",t),t.preventDefault(),t.stopPropagation()})),t.children.item(1).addEventListener((0,d.Le)(),(function(t){ge(e,"ir",t),t.preventDefault(),t.stopPropagation()})),t.children.item(2).addEventListener((0,d.Le)(),(function(t){ge(e,"sv",t),t.preventDefault(),t.stopPropagation()}))},t}(he),be=function(e,t){return(0,N.Gb)(e,t)?getSelection().toString():""},we=function(e,t){t.addEventListener("focus",(function(){e.options.focus&&e.options.focus(a(e)),v(e,["subToolbar","hint"])}))},Ee=function(e,t){t.addEventListener("dblclick",(function(t){"IMG"===t.target.tagName&&(0,B.E)(t.target,e.options.lang,e.options.theme)}))},ke=function(e,t){t.addEventListener("blur",(function(t){if("ir"===e.currentMode){var n=e.ir.element.querySelector(".vditor-ir__node--expand");n&&n.classList.remove("vditor-ir__node--expand")}else"wysiwyg"!==e.currentMode||e.wysiwyg.selectPopover.contains(t.relatedTarget)||e.wysiwyg.hideComment();e[e.currentMode].range=(0,N.zh)(e),e.options.blur&&e.options.blur(a(e))}))},Se=function(e,t){t.addEventListener("dragstart",(function(e){e.dataTransfer.setData(i.g.DROP_EDITOR,i.g.DROP_EDITOR)})),t.addEventListener("drop",(function(t){t.dataTransfer.getData(i.g.DROP_EDITOR)?lt(e):(t.dataTransfer.types.includes("Files")||t.dataTransfer.types.includes("text/html"))&&St(e,t,{pasteCode:function(e){document.execCommand("insertHTML",!1,e)}})}))},Ce=function(e,t,n){t.addEventListener("copy",(function(t){return n(t,e)}))},Le=function(e,t,n){t.addEventListener("cut",(function(t){n(t,e),e.options.comment.enable&&"wysiwyg"===e.currentMode&&e.wysiwyg.getComments(e),document.execCommand("delete")}))},Te=function(e){if("wysiwyg"===e.currentMode&&e.options.comment.enable&&e.options.comment.adjustTop(e.wysiwyg.getComments(e,!0)),e.options.typewriterMode){var t=e[e.currentMode].element,n=(0,N.Ny)(t).top;"string"!=typeof e.options.height||e.element.classList.contains("vditor--fullscreen")||window.scrollTo(window.scrollX,n+e.element.offsetTop+e.toolbar.element.offsetHeight-window.innerHeight/2+10),("number"==typeof e.options.height||e.element.classList.contains("vditor--fullscreen"))&&(t.scrollTop=n+t.scrollTop-t.clientHeight/2+10)}},Me=function(e,t){t.addEventListener("keydown",(function(t){if(!(e.options.hint.extend.length>1||e.toolbar.elements.emoji)||!e.hint.select(t,e)){if(e.options.comment.enable&&"wysiwyg"===e.currentMode&&("Backspace"===t.key||R("⌘X",t))&&e.wysiwyg.getComments(e),"sv"===e.currentMode){if(function(e,t){var n,r,i,o,a;if(e.sv.composingLock=t.isComposing,t.isComposing)return!1;if(-1!==t.key.indexOf("Arrow")||"Meta"===t.key||"Control"===t.key||"Alt"===t.key||"Shift"===t.key||"CapsLock"===t.key||"Escape"===t.key||/^F\d{1,2}$/.test(t.key)||e.undo.recordFirstPosition(e,t),"Enter"!==t.key&&"Tab"!==t.key&&"Backspace"!==t.key&&-1===t.key.indexOf("Arrow")&&!(0,d.yl)(t)&&"Escape"!==t.key)return!1;var l=(0,N.zh)(e),s=l.startContainer;3!==l.startContainer.nodeType&&"DIV"===l.startContainer.tagName&&(s=l.startContainer.childNodes[l.startOffset-1]);var c=(0,y.a1)(s,"data-type","text"),u=(0,y.a1)(s,"data-type","blockquote-marker");if(!u&&0===l.startOffset&&c&&c.previousElementSibling&&"blockquote-marker"===c.previousElementSibling.getAttribute("data-type")&&(u=c.previousElementSibling),u&&"Enter"===t.key&&!(0,d.yl)(t)&&!t.altKey&&""===u.nextElementSibling.textContent.trim()&&(0,N.im)(u,e.sv.element,l).start===u.textContent.length)return"padding"===(null===(n=u.previousElementSibling)||void 0===n?void 0:n.getAttribute("data-type"))&&u.previousElementSibling.setAttribute("data-action","enter-remove"),u.remove(),De(e),t.preventDefault(),!0;var p=(0,y.a1)(s,"data-type","li-marker"),m=(0,y.a1)(s,"data-type","task-marker"),f=p;if(f||m&&"task-marker"!==m.nextElementSibling.getAttribute("data-type")&&(f=m),f||0!==l.startOffset||!c||!c.previousElementSibling||"li-marker"!==c.previousElementSibling.getAttribute("data-type")&&"task-marker"!==c.previousElementSibling.getAttribute("data-type")||(f=c.previousElementSibling),f){var h=(0,N.im)(f,e.sv.element,l).start,v="task-marker"===f.getAttribute("data-type"),g=f;if(v&&(g=f.previousElementSibling.previousElementSibling.previousElementSibling),h===f.textContent.length){if("Enter"===t.key&&!(0,d.yl)(t)&&!t.altKey&&!t.shiftKey&&""===f.nextElementSibling.textContent.trim())return"padding"===(null===(r=g.previousElementSibling)||void 0===r?void 0:r.getAttribute("data-type"))?(g.previousElementSibling.remove(),q(e)):(v&&(g.remove(),f.previousElementSibling.previousElementSibling.remove(),f.previousElementSibling.remove()),f.nextElementSibling.remove(),f.remove(),De(e)),t.preventDefault(),!0;if("Tab"===t.key)return g.insertAdjacentHTML("beforebegin",'<span data-type="padding">'+g.textContent.replace(/\S/g," ")+"</span>"),/^\d/.test(g.textContent)&&(g.textContent=g.textContent.replace(/^\d{1,}/,"1"),l.selectNodeContents(f.firstChild),l.collapse(!1)),q(e),t.preventDefault(),!0}}if(dt(e,l,t))return!0;var w=(0,y.a1)(s,"data-block","0"),E=(0,b.S)(s,"SPAN");if("Enter"===t.key&&!(0,d.yl)(t)&&!t.altKey&&!t.shiftKey&&w){var k=!1,S=w.textContent.match(/^\n+/);(0,N.im)(w,e.sv.element).start<=(S?S[0].length:0)&&(k=!0);var C="\n";if(E){if("enter-remove"===(null===(i=E.previousElementSibling)||void 0===i?void 0:i.getAttribute("data-action")))return E.previousElementSibling.remove(),De(e),t.preventDefault(),!0;C+=Ne(E)}return l.insertNode(document.createTextNode(C)),l.collapse(!1),w&&""!==w.textContent.trim()&&!k?q(e):De(e),t.preventDefault(),!0}if("Backspace"===t.key&&!(0,d.yl)(t)&&!t.altKey&&!t.shiftKey){if(E&&"newline"===(null===(o=E.previousElementSibling)||void 0===o?void 0:o.getAttribute("data-type"))&&1===(0,N.im)(E,e.sv.element,l).start&&-1===E.getAttribute("data-type").indexOf("code-block-"))return l.setStart(E,0),l.extractContents(),""!==E.textContent.trim()?q(e):De(e),t.preventDefault(),!0;if(w&&0===(0,N.im)(w,e.sv.element,l).start&&w.previousElementSibling){l.extractContents();var L=w.previousElementSibling.lastElementChild;return"newline"===L.getAttribute("data-type")&&(L.remove(),L=w.previousElementSibling.lastElementChild),"newline"!==L.getAttribute("data-type")&&(L.insertAdjacentHTML("afterend",w.innerHTML),w.remove()),""===w.textContent.trim()||(null===(a=w.previousElementSibling)||void 0===a?void 0:a.querySelector('[data-type="code-block-open-marker"]'))?("newline"!==L.getAttribute("data-type")&&(l.selectNodeContents(L.lastChild),l.collapse(!1)),De(e)):q(e),t.preventDefault(),!0}}return!1}(e,t))return}else if("wysiwyg"===e.currentMode){if(function(e,t){if(e.wysiwyg.composingLock=t.isComposing,t.isComposing)return!1;-1!==t.key.indexOf("Arrow")||"Meta"===t.key||"Control"===t.key||"Alt"===t.key||"Shift"===t.key||"CapsLock"===t.key||"Escape"===t.key||/^F\d{1,2}$/.test(t.key)||e.undo.recordFirstPosition(e,t);var n=(0,N.zh)(e),r=n.startContainer;if(!Ke(t,e,r))return!1;if(Fe(n,e,t),Et(n),"Enter"!==t.key&&"Tab"!==t.key&&"Backspace"!==t.key&&-1===t.key.indexOf("Arrow")&&!(0,d.yl)(t)&&"Escape"!==t.key&&"Delete"!==t.key)return!1;var o=(0,y.F9)(r),a=(0,y.lG)(r,"P");if(ct(t,e,a,n))return!0;if(st(n,e,a,t))return!0;if(vt(e,t,n))return!0;var l=(0,y.fb)(r,"vditor-wysiwyg__block");if(l){if("Escape"===t.key&&2===l.children.length)return e.wysiwyg.popover.style.display="none",l.firstElementChild.style.display="none",e.wysiwyg.element.blur(),t.preventDefault(),!0;if(!(0,d.yl)(t)&&!t.shiftKey&&t.altKey&&"Enter"===t.key&&"code-block"===l.getAttribute("data-type")){var s=e.wysiwyg.popover.querySelector(".vditor-input");return s.focus(),s.select(),t.preventDefault(),!0}if("0"===l.getAttribute("data-block")){if(gt(e,t,l.firstElementChild,n))return!0;if($e(e,t,n,l.firstElementChild,l))return!0;if("yaml-front-matter"!==l.getAttribute("data-type")&&et(e,t,n,l.firstElementChild,l))return!0}}if(yt(e,n,t,a))return!0;var c=(0,y.E2)(r,"BLOCKQUOTE");if(c&&!t.shiftKey&&t.altKey&&"Enter"===t.key){(0,d.yl)(t)?n.setStartBefore(c):n.setStartAfter(c),(0,N.Hc)(n);var u=document.createElement("p");return u.setAttribute("data-block","0"),u.innerHTML="\n",n.insertNode(u),n.collapse(!0),(0,N.Hc)(n),X(e),Te(e),t.preventDefault(),!0}var p,m=(0,b.W)(r);if(m){if("H6"===m.tagName&&r.textContent.length===n.startOffset&&!(0,d.yl)(t)&&!t.shiftKey&&!t.altKey&&"Enter"===t.key){var f=document.createElement("p");return f.textContent="\n",f.setAttribute("data-block","0"),r.parentElement.insertAdjacentElement("afterend",f),n.setStart(f,0),(0,N.Hc)(n),X(e),Te(e),t.preventDefault(),!0}var h;if(R("⌘=",t))return(h=parseInt(m.tagName.substr(1),10)-1)>0&&(ee(e,"h"+h),X(e)),t.preventDefault(),!0;if(R("⌘-",t))return(h=parseInt(m.tagName.substr(1),10)+1)<7&&(ee(e,"h"+h),X(e)),t.preventDefault(),!0;"Backspace"!==t.key||(0,d.yl)(t)||t.shiftKey||t.altKey||1!==m.textContent.length||te(e)}if(bt(e,n,t))return!0;if(t.altKey&&"Enter"===t.key&&!(0,d.yl)(t)&&!t.shiftKey){var v=(0,y.lG)(r,"A"),g=(0,y.a1)(r,"data-type","link-ref"),w=(0,y.a1)(r,"data-type","footnotes-ref");if(v||g||w||m&&2===m.tagName.length){var E=e.wysiwyg.popover.querySelector("input");E.focus(),E.select()}}if(re(e,t))return!0;if(R("⇧⌘U",t)&&(p=e.wysiwyg.popover.querySelector('[data-type="up"]')))return p.click(),t.preventDefault(),!0;if(R("⇧⌘D",t)&&(p=e.wysiwyg.popover.querySelector('[data-type="down"]')))return p.click(),t.preventDefault(),!0;if(dt(e,n,t))return!0;if(!(0,d.yl)(t)&&t.shiftKey&&!t.altKey&&"Enter"===t.key&&"LI"!==r.parentElement.tagName&&"P"!==r.parentElement.tagName)return["STRONG","STRIKE","S","I","EM","B"].includes(r.parentElement.tagName)?n.insertNode(document.createTextNode("\n"+i.g.ZWSP)):n.insertNode(document.createTextNode("\n")),n.collapse(!1),(0,N.Hc)(n),X(e),Te(e),t.preventDefault(),!0;if("Backspace"===t.key&&!(0,d.yl)(t)&&!t.shiftKey&&!t.altKey&&""===n.toString()){if(wt(e,n,t,a))return!0;if(o){if(o.previousElementSibling&&o.previousElementSibling.classList.contains("vditor-wysiwyg__block")&&"0"===o.previousElementSibling.getAttribute("data-block")&&"UL"!==o.tagName&&"OL"!==o.tagName){var k=(0,N.im)(o,e.wysiwyg.element,n).start;if(0===k&&0===n.startOffset||1===k&&o.innerText.startsWith(i.g.ZWSP))return ne(o.previousElementSibling.lastElementChild,e,!1),""===o.innerHTML.trim().replace(i.g.ZWSP,"")&&(o.remove(),X(e)),t.preventDefault(),!0}var S=n.startOffset;if(""===n.toString()&&3===r.nodeType&&"\n"===r.textContent.charAt(S-2)&&r.textContent.charAt(S-1)!==i.g.ZWSP&&["STRONG","STRIKE","S","I","EM","B"].includes(r.parentElement.tagName))return r.textContent=r.textContent.substring(0,S-1)+i.g.ZWSP,n.setStart(r,S),n.collapse(!0),X(e),t.preventDefault(),!0;r.textContent===i.g.ZWSP&&1===n.startOffset&&!r.previousSibling&&function(e){for(var t=e.startContainer.nextSibling;t&&""===t.textContent;)t=t.nextSibling;return!(!t||3===t.nodeType||"CODE"!==t.tagName&&"math-inline"!==t.getAttribute("data-type")&&"html-entity"!==t.getAttribute("data-type")&&"html-inline"!==t.getAttribute("data-type"))}(n)&&(r.textContent=""),o.querySelectorAll("span.vditor-wysiwyg__block[data-type='math-inline']").forEach((function(e){e.firstElementChild.style.display="inline",e.lastElementChild.style.display="none"})),o.querySelectorAll("span.vditor-wysiwyg__block[data-type='html-entity']").forEach((function(e){e.firstElementChild.style.display="inline",e.lastElementChild.style.display="none"}))}}if((0,d.vU)()&&1===n.startOffset&&r.textContent.indexOf(i.g.ZWSP)>-1&&r.previousSibling&&3!==r.previousSibling.nodeType&&"CODE"===r.previousSibling.tagName&&("Backspace"===t.key||"ArrowLeft"===t.key))return n.selectNodeContents(r.previousSibling),n.collapse(!1),t.preventDefault(),!0;if(kt(t,o,n))return t.preventDefault(),!0;if(Ze(n,t.key),"ArrowDown"===t.key){var C=r.nextSibling;C&&3!==C.nodeType&&"math-inline"===C.getAttribute("data-type")&&n.setStartAfter(C)}return!(!o||!I(o,e,t,n)||(t.preventDefault(),0))}(e,t))return}else if("ir"===e.currentMode&&function(e,t){if(e.ir.composingLock=t.isComposing,t.isComposing)return!1;-1!==t.key.indexOf("Arrow")||"Meta"===t.key||"Control"===t.key||"Alt"===t.key||"Shift"===t.key||"CapsLock"===t.key||"Escape"===t.key||/^F\d{1,2}$/.test(t.key)||e.undo.recordFirstPosition(e,t);var n=(0,N.zh)(e),r=n.startContainer;if(!Ke(t,e,r))return!1;if(Fe(n,e,t),Et(n),"Enter"!==t.key&&"Tab"!==t.key&&"Backspace"!==t.key&&-1===t.key.indexOf("Arrow")&&!(0,d.yl)(t)&&"Escape"!==t.key&&"Delete"!==t.key)return!1;var o=(0,y.a1)(r,"data-newline","1");if(!(0,d.yl)(t)&&!t.altKey&&!t.shiftKey&&"Enter"===t.key&&o&&n.startOffset<o.textContent.length){var a=o.previousElementSibling;a&&(n.insertNode(document.createTextNode(a.textContent)),n.collapse(!1));var l=o.nextSibling;l&&(n.insertNode(document.createTextNode(l.textContent)),n.collapse(!0))}var s=(0,y.lG)(r,"P");if(ct(t,e,s,n))return!0;if(st(n,e,s,t))return!0;if(yt(e,n,t,s))return!0;var c=(0,y.fb)(r,"vditor-ir__marker--pre");if(c&&"PRE"===c.tagName){var u=c.firstChild;if(gt(e,t,c,n))return!0;if(("math-block"===u.getAttribute("data-type")||"html-block"===u.getAttribute("data-type"))&&et(e,t,n,u,c.parentElement))return!0;if($e(e,t,n,u,c.parentElement))return!0}var p=(0,y.a1)(r,"data-type","code-block-info");if(p){if("Enter"===t.key||"Tab"===t.key)return n.selectNodeContents(p.nextElementSibling.firstChild),n.collapse(!0),t.preventDefault(),v(e,["hint"]),!0;if("Backspace"===t.key){var m=(0,N.im)(p,e.ir.element).start;1===m&&n.setStart(r,0),2===m&&(e.hint.recentLanguage="")}if(et(e,t,n,p,p.parentElement))return v(e,["hint"]),!0}var f=(0,y.lG)(r,"TD")||(0,y.lG)(r,"TH");if(t.key.indexOf("Arrow")>-1&&f){var h=Xe(f);if(h&&et(e,t,n,f,h))return!0;var g=Ye(f);if(g&&$e(e,t,n,f,g))return!0}if(vt(e,t,n))return!0;if(bt(e,n,t))return!0;if(dt(e,n,t))return!0;var w=(0,b.W)(r);if(w){var E;if(R("⌘=",t))return(E=w.querySelector(".vditor-ir__marker--heading"))&&E.textContent.trim().length>1&&Tt(e,E.textContent.substr(1)),t.preventDefault(),!0;if(R("⌘-",t))return(E=w.querySelector(".vditor-ir__marker--heading"))&&E.textContent.trim().length<6&&Tt(e,E.textContent.trim()+"# "),t.preventDefault(),!0}var k=(0,y.F9)(r);if("Backspace"===t.key&&!(0,d.yl)(t)&&!t.shiftKey&&!t.altKey&&""===n.toString()){if(wt(e,n,t,s))return!0;if(k&&k.previousElementSibling&&"UL"!==k.tagName&&"OL"!==k.tagName&&("code-block"===k.previousElementSibling.getAttribute("data-type")||"math-block"===k.previousElementSibling.getAttribute("data-type"))){var S=(0,N.im)(k,e.ir.element,n).start;if(0===S||1===S&&k.innerText.startsWith(i.g.ZWSP))return n.selectNodeContents(k.previousElementSibling.querySelector(".vditor-ir__marker--pre code")),n.collapse(!1),P(n,e),""===k.textContent.trim().replace(i.g.ZWSP,"")&&(k.remove(),Lt(e)),t.preventDefault(),!0}if(w){var C=w.firstElementChild.textContent.length;(0,N.im)(w,e.ir.element).start===C&&(n.setStart(w.firstElementChild.firstChild,C-1),n.collapse(!0),(0,N.Hc)(n))}}return!(("ArrowUp"!==t.key&&"ArrowDown"!==t.key||!k||(k.querySelectorAll(".vditor-ir__node").forEach((function(e){e.contains(r)||e.classList.add("vditor-ir__node--hidden")})),!kt(t,k,n)))&&(Ze(n,t.key),!k||!I(k,e,t,n)||(t.preventDefault(),0)))}(e,t))return;if(e.options.ctrlEnter&&R("⌘Enter",t))return e.options.ctrlEnter(a(e)),void t.preventDefault();if(R("⌘Z",t)&&!e.toolbar.elements.undo)return e.undo.undo(e),void t.preventDefault();if(R("⌘Y",t)&&!e.toolbar.elements.redo)return e.undo.redo(e),void t.preventDefault();if("Escape"===t.key)return"block"===e.hint.element.style.display?e.hint.element.style.display="none":e.options.esc&&!t.isComposing&&e.options.esc(a(e)),void t.preventDefault();if((0,d.yl)(t)&&t.altKey&&!t.shiftKey&&/^Digit[1-6]$/.test(t.code)){if("wysiwyg"===e.currentMode){var n=t.code.replace("Digit","H");(0,y.lG)(getSelection().getRangeAt(0).startContainer,n)?te(e):ee(e,n),X(e)}else"sv"===e.currentMode?Oe(e,"#".repeat(parseInt(t.code.replace("Digit",""),10))+" "):"ir"===e.currentMode&&Tt(e,"#".repeat(parseInt(t.code.replace("Digit",""),10))+" ");return t.preventDefault(),!0}if((0,d.yl)(t)&&t.altKey&&!t.shiftKey&&/^Digit[7-9]$/.test(t.code))return"Digit7"===t.code?ge(e,"wysiwyg",t):"Digit8"===t.code?ge(e,"ir",t):"Digit9"===t.code&&ge(e,"sv",t),!0;e.options.toolbar.find((function(n){return!n.hotkey||n.toolbar?!!n.toolbar&&!!n.toolbar.find((function(n){return!!n.hotkey&&(R(n.hotkey,t)?(e.toolbar.elements[n.name].children[0].dispatchEvent(new CustomEvent((0,d.Le)())),t.preventDefault(),!0):void 0)})):R(n.hotkey,t)?(e.toolbar.elements[n.name].children[0].dispatchEvent(new CustomEvent((0,d.Le)())),t.preventDefault(),!0):void 0}))}}))},Ae=function(e,t){t.addEventListener("selectstart",(function(n){t.onmouseup=function(){setTimeout((function(){var t=be(e[e.currentMode].element);t.trim()?("wysiwyg"===e.currentMode&&e.options.comment.enable&&((0,y.a1)(n.target,"data-type","footnotes-block")||(0,y.a1)(n.target,"data-type","link-ref-defs-block")?e.wysiwyg.hideComment():e.wysiwyg.showComment()),e.options.select&&e.options.select(t)):"wysiwyg"===e.currentMode&&e.options.comment.enable&&e.wysiwyg.hideComment()}))}}))},_e=function(e,t){var n=(0,N.zh)(e);n.extractContents(),n.insertNode(document.createTextNode(Lute.Caret)),n.insertNode(document.createTextNode(t));var r=(0,y.a1)(n.startContainer,"data-block","0");r||(r=e.sv.element);var i="<div data-block='0'>"+e.lute.Md2VditorSVDOM(r.textContent).replace(/<span data-type="newline"><br \/><span style="display: none">\n<\/span><\/span><span data-type="newline"><br \/><span style="display: none">\n<\/span><\/span></g,'<span data-type="newline"><br /><span style="display: none">\n</span></span><span data-type="newline"><br /><span style="display: none">\n</span></span></div><div data-block="0"><')+"</div>";r.isEqualNode(e.sv.element)?r.innerHTML=i:r.outerHTML=i,(0,N.ib)(e.sv.element,n),Te(e)},xe=function(e,t,n){void 0===n&&(n=!0);var r=e;for(3===r.nodeType&&(r=r.parentElement);r;){if(r.getAttribute("data-type")===t)return r;r=n?r.previousElementSibling:r.nextElementSibling}return!1},He=function(e,t){return w("SpinVditorSVDOM",e,"argument",t.options.debugger),e="<div data-block='0'>"+t.lute.SpinVditorSVDOM(e).replace(/<span data-type="newline"><br \/><span style="display: none">\n<\/span><\/span><span data-type="newline"><br \/><span style="display: none">\n<\/span><\/span></g,'<span data-type="newline"><br /><span style="display: none">\n</span></span><span data-type="newline"><br /><span style="display: none">\n</span></span></div><div data-block="0"><')+"</div>",w("SpinVditorSVDOM",e,"result",t.options.debugger),e},Ne=function(e){var t=e.getAttribute("data-type"),n=e.previousElementSibling,r=t&&"text"!==t&&"table"!==t&&"heading-marker"!==t&&"newline"!==t&&"yaml-front-matter-open-marker"!==t&&"yaml-front-matter-close-marker"!==t&&"code-block-info"!==t&&"code-block-close-marker"!==t&&"code-block-open-marker"!==t?e.textContent:"",i=!1;for("newline"===t&&(i=!0);n&&!i;){var o=n.getAttribute("data-type");if("li-marker"===o||"blockquote-marker"===o||"task-marker"===o||"padding"===o){var a=n.textContent;if("li-marker"!==o||"code-block-open-marker"!==t&&"code-block-info"!==t)if("code-block-close-marker"===t&&n.nextElementSibling.isSameNode(e)){var l=xe(e,"code-block-open-marker");l&&l.previousElementSibling&&(n=l.previousElementSibling,r=a+r)}else r=a+r;else r=a.replace(/\S/g," ")+r}else"newline"===o&&(i=!0);n=n.previousElementSibling}return r},De=function(e,t){void 0===t&&(t={enableAddUndoStack:!0,enableHint:!1,enableInput:!0}),t.enableHint&&e.hint.render(e),e.preview.render(e);var n=a(e);"function"==typeof e.options.input&&t.enableInput&&e.options.input(n),e.options.counter.enable&&e.counter.render(e,n),e.options.cache.enable&&(0,d.pK)()&&(localStorage.setItem(e.options.cache.id,n),e.options.cache.after&&e.options.cache.after(n)),e.devtools&&e.devtools.renderEchart(e),clearTimeout(e.sv.processTimeoutId),e.sv.processTimeoutId=window.setTimeout((function(){t.enableAddUndoStack&&!e.sv.composingLock&&e.undo.addToUndoStack(e)}),e.options.undoDelay)},Oe=function(e,t){var n=(0,N.zh)(e),r=(0,b.S)(n.startContainer,"SPAN");r&&""!==r.textContent.trim()&&(t="\n"+t),n.collapse(!0),document.execCommand("insertHTML",!1,t)},Ie=function(e,t,n,r){var i=(0,N.zh)(e),o=t.getAttribute("data-type");0===e.sv.element.childNodes.length&&(e.sv.element.innerHTML='<span data-type="p" data-block="0"><span data-type="text"><wbr></span></span><span data-type="newline"><br><span style="display: none">\n</span></span>',(0,N.ib)(e.sv.element,i));var a=(0,y.F9)(i.startContainer),l=(0,b.S)(i.startContainer,"SPAN");if(a){if("link"===o){var s=void 0;return s=""===i.toString()?""+n+Lute.Caret+r:""+n+i.toString()+r.replace(")",Lute.Caret+")"),void document.execCommand("insertHTML",!1,s)}if("italic"===o||"bold"===o||"strike"===o||"inline-code"===o||"code"===o||"table"===o||"line"===o){s=void 0;return s=""===i.toString()?""+n+Lute.Caret+("code"===o?"":r):""+n+i.toString()+Lute.Caret+("code"===o?"":r),"table"===o||"code"===o&&l&&""!==l.textContent?s="\n\n"+s:"line"===o&&(s="\n\n"+n+"\n"+Lute.Caret),void document.execCommand("insertHTML",!1,s)}if(("check"===o||"list"===o||"ordered-list"===o||"quote"===o)&&l){var d="* ";"check"===o?d="* [ ] ":"ordered-list"===o?d="1. ":"quote"===o&&(d="> ");var c=xe(l,"newline");return c?c.insertAdjacentText("afterend",d):a.insertAdjacentText("afterbegin",d),void q(e)}(0,N.ib)(e.sv.element,i),De(e)}},je=function(e){switch(e.currentMode){case"ir":return e.ir.element;case"wysiwyg":return e.wysiwyg.element;case"sv":return e.sv.element}},Re=function(e,t){e.options.upload.setHeaders&&(e.options.upload.headers=e.options.upload.setHeaders()),e.options.upload.headers&&Object.keys(e.options.upload.headers).forEach((function(n){t.setRequestHeader(n,e.options.upload.headers[n])}))},Pe=function(e,t,n,r){return new(n||(n=Promise))((function(i,o){function a(e){try{s(r.next(e))}catch(e){o(e)}}function l(e){try{s(r.throw(e))}catch(e){o(e)}}function s(e){var t;e.done?i(e.value):(t=e.value,t instanceof n?t:new n((function(e){e(t)}))).then(a,l)}s((r=r.apply(e,t||[])).next())}))},Be=function(e,t){var n,r,i,o,a={label:0,sent:function(){if(1&i[0])throw i[1];return i[1]},trys:[],ops:[]};return o={next:l(0),throw:l(1),return:l(2)},"function"==typeof Symbol&&(o[Symbol.iterator]=function(){return this}),o;function l(o){return function(l){return function(o){if(n)throw new TypeError("Generator is already executing.");for(;a;)try{if(n=1,r&&(i=2&o[0]?r.return:o[0]?r.throw||((i=r.return)&&i.call(r),0):r.next)&&!(i=i.call(r,o[1])).done)return i;switch(r=0,i&&(o=[2&o[0],i.value]),o[0]){case 0:case 1:i=o;break;case 4:return a.label++,{value:o[1],done:!1};case 5:a.label++,r=o[1],o=[0];continue;case 7:o=a.ops.pop(),a.trys.pop();continue;default:if(!(i=a.trys,(i=i.length>0&&i[i.length-1])||6!==o[0]&&2!==o[0])){a=0;continue}if(3===o[0]&&(!i||o[1]>i[0]&&o[1]<i[3])){a.label=o[1];break}if(6===o[0]&&a.label<i[1]){a.label=i[1],i=o;break}if(i&&a.label<i[2]){a.label=i[2],a.ops.push(o);break}i[2]&&a.ops.pop(),a.trys.pop();continue}o=t.call(e,a)}catch(e){o=[6,e],r=0}finally{n=i=0}if(5&o[0])throw o[1];return{value:o[0]?o[1]:void 0,done:!0}}([o,l])}}},qe=function(){this.isUploading=!1,this.element=document.createElement("div"),this.element.className="vditor-upload"},Ve=function(e,t,n){return Pe(void 0,void 0,void 0,(function(){var r,i,o,a,l,s,d,c,u,p,m,f,h,v;return Be(this,(function(g){switch(g.label){case 0:for(r=[],i=!0===e.options.upload.multiple?t.length:1,f=0;f<i;f++)(o=t[f])instanceof DataTransferItem&&(o=o.getAsFile()),r.push(o);return e.options.upload.handler?[4,e.options.upload.handler(r)]:[3,2];case 1:return"string"==typeof(a=g.sent())?(e.tip.show(a),[2]):[2];case 2:return e.options.upload.url&&e.upload?e.options.upload.file?[4,e.options.upload.file(r)]:[3,4]:(n&&(n.value=""),e.tip.show("please config: options.upload.url"),[2]);case 3:r=g.sent(),g.label=4;case 4:if(e.options.upload.validate&&"string"==typeof(a=e.options.upload.validate(r)))return e.tip.show(a),[2];if(l=je(e),e.upload.range=(0,N.zh)(e),s=function(e,t){e.tip.hide();for(var n=[],r="",i="",o=(e.options.lang,e.options,function(o,a){var l=t[a],s=!0;l.name||(r+="<li>"+window.VditorI18n.nameEmpty+"</li>",s=!1),l.size>e.options.upload.max&&(r+="<li>"+l.name+" "+window.VditorI18n.over+" "+e.options.upload.max/1024/1024+"M</li>",s=!1);var d=l.name.lastIndexOf("."),c=l.name.substr(d),u=e.options.upload.filename(l.name.substr(0,d))+c;e.options.upload.accept&&(e.options.upload.accept.split(",").some((function(e){var t=e.trim();if(0===t.indexOf(".")){if(c.toLowerCase()===t.toLowerCase())return!0}else if(l.type.split("/")[0]===t.split("/")[0])return!0;return!1}))||(r+="<li>"+l.name+" "+window.VditorI18n.fileTypeError+"</li>",s=!1)),s&&(n.push(l),i+="<li>"+u+" "+window.VditorI18n.uploading+"</li>")}),a=t.length,l=0;l<a;l++)o(0,l);return e.tip.show("<ul>"+r+i+"</ul>"),n}(e,r),0===s.length)return n&&(n.value=""),[2];for(d=new FormData,c=e.options.upload.extraData,u=0,p=Object.keys(c);u<p.length;u++)m=p[u],d.append(m,c[m]);for(f=0,h=s.length;f<h;f++)d.append(e.options.upload.fieldName,s[f]);return(v=new XMLHttpRequest).open("POST",e.options.upload.url),e.options.upload.token&&v.setRequestHeader("X-Upload-Token",e.options.upload.token),e.options.upload.withCredentials&&(v.withCredentials=!0),Re(e,v),e.upload.isUploading=!0,l.setAttribute("contenteditable","false"),v.onreadystatechange=function(){if(v.readyState===XMLHttpRequest.DONE){if(e.upload.isUploading=!1,l.setAttribute("contenteditable","true"),v.status>=200&&v.status<300)if(e.options.upload.success)e.options.upload.success(l,v.responseText);else{var r=v.responseText;e.options.upload.format&&(r=e.options.upload.format(t,v.responseText)),function(e,t){je(t).focus();var n=JSON.parse(e),r="";1===n.code&&(r=""+n.msg),n.data.errFiles&&n.data.errFiles.length>0&&(r="<ul><li>"+r+"</li>",n.data.errFiles.forEach((function(e){var n=e.lastIndexOf("."),i=t.options.upload.filename(e.substr(0,n))+e.substr(n);r+="<li>"+i+" "+window.VditorI18n.uploadError+"</li>"})),r+="</ul>"),r?t.tip.show(r):t.tip.hide();var i="";Object.keys(n.data.succMap).forEach((function(e){var r=n.data.succMap[e],o=e.lastIndexOf("."),a=e.substr(o),l=t.options.upload.filename(e.substr(0,o))+a;0===(a=a.toLowerCase()).indexOf(".wav")||0===a.indexOf(".mp3")||0===a.indexOf(".ogg")?"wysiwyg"===t.currentMode?i+='<div class="vditor-wysiwyg__block" data-type="html-block"\n data-block="0"><pre><code>&lt;audio controls="controls" src="'+r+'"&gt;&lt;/audio&gt;</code></pre>\n':"ir"===t.currentMode?i+='<audio controls="controls" src="'+r+'"></audio>\n':i+="["+l+"]("+r+")\n":0===a.indexOf(".apng")||0===a.indexOf(".bmp")||0===a.indexOf(".gif")||0===a.indexOf(".ico")||0===a.indexOf(".cur")||0===a.indexOf(".jpg")||0===a.indexOf(".jpeg")||0===a.indexOf(".jfif")||0===a.indexOf(".pjp")||0===a.indexOf(".pjpeg")||0===a.indexOf(".png")||0===a.indexOf(".svg")||0===a.indexOf(".webp")?"wysiwyg"===t.currentMode?i+='<img alt="'+l+'" src="'+r+'">\n':i+="!["+l+"]("+r+")\n":"wysiwyg"===t.currentMode?i+='<a href="'+r+'">'+l+"</a>\n":i+="["+l+"]("+r+")\n"})),(0,N.Hc)(t.upload.range),document.execCommand("insertHTML",!1,i),t.upload.range=getSelection().getRangeAt(0).cloneRange()}(r,e)}else e.options.upload.error?e.options.upload.error(v.responseText):e.tip.show(v.responseText);n&&(n.value=""),e.upload.element.style.display="none"}},v.upload.onprogress=function(t){if(t.lengthComputable){var n=t.loaded/t.total*100;e.upload.element.style.display="block",e.upload.element.style.width=n+"%"}},v.send(d),[2]}}))}))},Ue=function(e,t,n){var r,o=(0,y.F9)(t.startContainer);if(o||(o=e.wysiwyg.element),n&&"formatItalic"!==n.inputType&&"deleteByDrag"!==n.inputType&&"insertFromDrop"!==n.inputType&&"formatBold"!==n.inputType&&"formatRemove"!==n.inputType&&"formatStrikeThrough"!==n.inputType&&"insertUnorderedList"!==n.inputType&&"insertOrderedList"!==n.inputType&&"formatOutdent"!==n.inputType&&"formatIndent"!==n.inputType&&""!==n.inputType||!n){var a=function(e){for(var t=e.previousSibling;t;){if(3!==t.nodeType&&"A"===t.tagName&&!t.previousSibling&&""===t.innerHTML.replace(i.g.ZWSP,"")&&t.nextSibling)return t;t=t.previousSibling}return!1}(t.startContainer);a&&a.remove(),e.wysiwyg.element.querySelectorAll("wbr").forEach((function(e){e.remove()})),t.insertNode(document.createElement("wbr")),o.querySelectorAll("[style]").forEach((function(e){e.removeAttribute("style")})),o.querySelectorAll(".vditor-comment").forEach((function(e){""===e.textContent.trim()&&(e.classList.remove("vditor-comment","vditor-comment--focus"),e.removeAttribute("data-cmtids"))})),null===(r=o.previousElementSibling)||void 0===r||r.querySelectorAll(".vditor-comment").forEach((function(e){""===e.textContent.trim()&&(e.classList.remove("vditor-comment","vditor-comment--focus"),e.removeAttribute("data-cmtids"))}));var l="";"link-ref-defs-block"===o.getAttribute("data-type")&&(o=e.wysiwyg.element);var s,d=o.isEqualNode(e.wysiwyg.element),c=(0,y.a1)(o,"data-type","footnotes-block");if(d)l=o.innerHTML;else{var u=(0,y.O9)(t.startContainer);if(u&&!c){var p=(0,b.S)(t.startContainer,"BLOCKQUOTE");o=p?(0,y.F9)(t.startContainer)||o:u}if(c&&(o=c),l=o.outerHTML,"UL"===o.tagName||"OL"===o.tagName){var m=o.previousElementSibling,f=o.nextElementSibling;!m||"UL"!==m.tagName&&"OL"!==m.tagName||(l=m.outerHTML+l,m.remove()),!f||"UL"!==f.tagName&&"OL"!==f.tagName||(l+=f.outerHTML,f.remove()),l=l.replace("<div><wbr><br></div>","<li><p><wbr><br></p></li>")}e.wysiwyg.element.querySelectorAll("[data-type='link-ref-defs-block']").forEach((function(e){e&&!o.isEqualNode(e)&&(l+=e.outerHTML,e.remove())})),e.wysiwyg.element.querySelectorAll("[data-type='footnotes-block']").forEach((function(e){e&&!o.isEqualNode(e)&&(l+=e.outerHTML,e.remove())}))}if('<p data-block="0">```<wbr></p>'===(l=l.replace(/<\/(strong|b)><strong data-marker="\W{2}">/g,"").replace(/<\/(em|i)><em data-marker="\W{1}">/g,"").replace(/<\/(s|strike)><s data-marker="~{1,2}">/g,""))&&e.hint.recentLanguage&&(l='<p data-block="0">```<wbr></p>'.replace("```","```"+e.hint.recentLanguage)),w("SpinVditorDOM",l,"argument",e.options.debugger),l=e.lute.SpinVditorDOM(l),w("SpinVditorDOM",l,"result",e.options.debugger),d)o.innerHTML=l;else if(o.outerHTML=l,c){var h=(0,y.E2)(e.wysiwyg.element.querySelector("wbr"),"LI");if(h){var v=e.wysiwyg.element.querySelector('sup[data-type="footnotes-ref"][data-footnotes-label="'+h.getAttribute("data-marker")+'"]');v&&v.setAttribute("aria-label",h.textContent.trim().substr(0,24))}}var g,E=e.wysiwyg.element.querySelectorAll("[data-type='link-ref-defs-block']");E.forEach((function(e,t){0===t?s=e:(s.insertAdjacentHTML("beforeend",e.innerHTML),e.remove())})),E.length>0&&e.wysiwyg.element.insertAdjacentElement("beforeend",E[0]);var k=e.wysiwyg.element.querySelectorAll("[data-type='footnotes-block']");k.forEach((function(e,t){0===t?g=e:(g.insertAdjacentHTML("beforeend",e.innerHTML),e.remove())})),k.length>0&&e.wysiwyg.element.insertAdjacentElement("beforeend",k[0]),(0,N.ib)(e.wysiwyg.element,t),e.wysiwyg.element.querySelectorAll(".vditor-wysiwyg__preview[data-render='2']").forEach((function(t){H(t,e)})),n&&("deleteContentBackward"===n.inputType||"deleteContentForward"===n.inputType)&&e.options.comment.enable&&(e.wysiwyg.triggerRemoveComment(e),e.options.comment.adjustTop(e.wysiwyg.getComments(e,!0)))}D(e),X(e,{enableAddUndoStack:!0,enableHint:!0,enableInput:!0})},We=function(e,t){return Object.defineProperty?Object.defineProperty(e,"raw",{value:t}):e.raw=t,e},ze=function(e,t,n,r){return new(n||(n=Promise))((function(i,o){function a(e){try{s(r.next(e))}catch(e){o(e)}}function l(e){try{s(r.throw(e))}catch(e){o(e)}}function s(e){var t;e.done?i(e.value):(t=e.value,t instanceof n?t:new n((function(e){e(t)}))).then(a,l)}s((r=r.apply(e,t||[])).next())}))},Ge=function(e,t){var n,r,i,o,a={label:0,sent:function(){if(1&i[0])throw i[1];return i[1]},trys:[],ops:[]};return o={next:l(0),throw:l(1),return:l(2)},"function"==typeof Symbol&&(o[Symbol.iterator]=function(){return this}),o;function l(o){return function(l){return function(o){if(n)throw new TypeError("Generator is already executing.");for(;a;)try{if(n=1,r&&(i=2&o[0]?r.return:o[0]?r.throw||((i=r.return)&&i.call(r),0):r.next)&&!(i=i.call(r,o[1])).done)return i;switch(r=0,i&&(o=[2&o[0],i.value]),o[0]){case 0:case 1:i=o;break;case 4:return a.label++,{value:o[1],done:!1};case 5:a.label++,r=o[1],o=[0];continue;case 7:o=a.ops.pop(),a.trys.pop();continue;default:if(!(i=a.trys,(i=i.length>0&&i[i.length-1])||6!==o[0]&&2!==o[0])){a=0;continue}if(3===o[0]&&(!i||o[1]>i[0]&&o[1]<i[3])){a.label=o[1];break}if(6===o[0]&&a.label<i[1]){a.label=i[1],i=o;break}if(i&&a.label<i[2]){a.label=i[2],a.ops.push(o);break}i[2]&&a.ops.pop(),a.trys.pop();continue}o=t.call(e,a)}catch(e){o=[6,e],r=0}finally{n=i=0}if(5&o[0])throw o[1];return{value:o[0]?o[1]:void 0,done:!0}}([o,l])}}},Ke=function(e,t,n){if(229===e.keyCode&&""===e.code&&"Unidentified"===e.key&&"sv"!==t.currentMode){var r=(0,y.F9)(n);if(r&&""===r.textContent.trim())return t[t.currentMode].composingLock=!0,!1}return!0},Fe=function(e,t,n){if(!("Enter"===n.key||"Tab"===n.key||"Backspace"===n.key||n.key.indexOf("Arrow")>-1||(0,d.yl)(n)||"Escape"===n.key||n.shiftKey||n.altKey)){var r=(0,y.lG)(e.startContainer,"P")||(0,y.lG)(e.startContainer,"LI");if(r&&0===(0,N.im)(r,t[t.currentMode].element,e).start){var o=document.createTextNode(i.g.ZWSP);e.insertNode(o),e.setStartAfter(o)}}},Ze=function(e,t){if("ArrowDown"===t||"ArrowUp"===t){var n=(0,y.a1)(e.startContainer,"data-type","math-inline")||(0,y.a1)(e.startContainer,"data-type","html-entity")||(0,y.a1)(e.startContainer,"data-type","html-inline");n&&("ArrowDown"===t&&e.setStartAfter(n.parentElement),"ArrowUp"===t&&e.setStartBefore(n.parentElement))}},Je=function(e,t){var n=(0,N.zh)(e),r=(0,y.F9)(n.startContainer);r&&(r.insertAdjacentHTML(t,'<p data-block="0">'+i.g.ZWSP+"<wbr>\n</p>"),(0,N.ib)(e[e.currentMode].element,n),pe(e),lt(e))},Xe=function(e){var t=(0,y.lG)(e,"TABLE");return!(!t||!t.rows[0].cells[0].isSameNode(e))&&t},Ye=function(e){var t=(0,y.lG)(e,"TABLE");return!(!t||!t.lastElementChild.lastElementChild.lastElementChild.isSameNode(e))&&t},Qe=function(e,t,n){void 0===n&&(n=!0);var r=e.previousElementSibling;return r||(r=e.parentElement.previousElementSibling?e.parentElement.previousElementSibling.lastElementChild:"TBODY"===e.parentElement.parentElement.tagName&&e.parentElement.parentElement.previousElementSibling?e.parentElement.parentElement.previousElementSibling.lastElementChild.lastElementChild:null),r&&(t.selectNodeContents(r),n||t.collapse(!1),(0,N.Hc)(t)),r},$e=function(e,t,n,r,o){var a=(0,N.im)(r,e[e.currentMode].element,n);if("ArrowDown"===t.key&&-1===r.textContent.trimRight().substr(a.start).indexOf("\n")||"ArrowRight"===t.key&&a.start>=r.textContent.trimRight().length){var l=o.nextElementSibling;return!l||l&&("TABLE"===l.tagName||l.getAttribute("data-type"))?(o.insertAdjacentHTML("afterend",'<p data-block="0">'+i.g.ZWSP+"<wbr></p>"),(0,N.ib)(e[e.currentMode].element,n)):(n.selectNodeContents(l),n.collapse(!0),(0,N.Hc)(n)),t.preventDefault(),!0}return!1},et=function(e,t,n,r,o){var a=(0,N.im)(r,e[e.currentMode].element,n);if("ArrowUp"===t.key&&-1===r.textContent.substr(0,a.start).indexOf("\n")||("ArrowLeft"===t.key||"Backspace"===t.key&&""===n.toString())&&0===a.start){var l=o.previousElementSibling;return!l||l&&("TABLE"===l.tagName||l.getAttribute("data-type"))?(o.insertAdjacentHTML("beforebegin",'<p data-block="0">'+i.g.ZWSP+"<wbr></p>"),(0,N.ib)(e[e.currentMode].element,n)):(n.selectNodeContents(l),n.collapse(!1),(0,N.Hc)(n)),t.preventDefault(),!0}return!1},tt=function(e,t,n,r){void 0===r&&(r=!0);var i=(0,y.lG)(t.startContainer,"LI");if(e[e.currentMode].element.querySelectorAll("wbr").forEach((function(e){e.remove()})),t.insertNode(document.createElement("wbr")),r&&i){for(var o="",a=0;a<i.parentElement.childElementCount;a++){var l=i.parentElement.children[a].querySelector("input");l&&l.remove(),o+='<p data-block="0">'+i.parentElement.children[a].innerHTML.trimLeft()+"</p>"}i.parentElement.insertAdjacentHTML("beforebegin",o),i.parentElement.remove()}else if(i)if("check"===n)i.parentElement.querySelectorAll("li").forEach((function(e){e.insertAdjacentHTML("afterbegin",'<input type="checkbox" />'+(0===e.textContent.indexOf(" ")?"":" ")),e.classList.add("vditor-task")}));else{i.querySelector("input")&&i.parentElement.querySelectorAll("li").forEach((function(e){e.querySelector("input").remove(),e.classList.remove("vditor-task")}));var s=void 0;"list"===n?(s=document.createElement("ul")).setAttribute("data-marker","*"):(s=document.createElement("ol")).setAttribute("data-marker","1."),s.setAttribute("data-block","0"),s.setAttribute("data-tight",i.parentElement.getAttribute("data-tight")),s.innerHTML=i.parentElement.innerHTML,i.parentElement.parentNode.replaceChild(s,i.parentElement)}else{var d=(0,y.a1)(t.startContainer,"data-block","0");d||(e[e.currentMode].element.querySelector("wbr").remove(),(d=e[e.currentMode].element.querySelector("p")).innerHTML="<wbr>"),"check"===n?(d.insertAdjacentHTML("beforebegin",'<ul data-block="0"><li class="vditor-task"><input type="checkbox" /> '+d.innerHTML+"</li></ul>"),d.remove()):"list"===n?(d.insertAdjacentHTML("beforebegin",'<ul data-block="0"><li>'+d.innerHTML+"</li></ul>"),d.remove()):"ordered-list"===n&&(d.insertAdjacentHTML("beforebegin",'<ol data-block="0"><li>'+d.innerHTML+"</li></ol>"),d.remove())}},nt=function(e,t,n){var r=t.previousElementSibling;if(t&&r){var i=[t];Array.from(n.cloneContents().children).forEach((function(e,n){3!==e.nodeType&&t&&""!==e.textContent.trim()&&t.getAttribute("data-node-id")===e.getAttribute("data-node-id")&&(0!==n&&i.push(t),t=t.nextElementSibling)})),e[e.currentMode].element.querySelectorAll("wbr").forEach((function(e){e.remove()})),n.insertNode(document.createElement("wbr"));var o=r.parentElement,a="";i.forEach((function(e){var t=e.getAttribute("data-marker");1!==t.length&&(t="1"+t.slice(-1)),a+='<li data-node-id="'+e.getAttribute("data-node-id")+'" data-marker="'+t+'">'+e.innerHTML+"</li>",e.remove()})),r.insertAdjacentHTML("beforeend","<"+o.tagName+' data-block="0">'+a+"</"+o.tagName+">"),"wysiwyg"===e.currentMode?o.outerHTML=e.lute.SpinVditorDOM(o.outerHTML):o.outerHTML=e.lute.SpinVditorIRDOM(o.outerHTML),(0,N.ib)(e[e.currentMode].element,n);var l=(0,y.O9)(n.startContainer);l&&l.querySelectorAll(".vditor-"+e.currentMode+"__preview[data-render='2']").forEach((function(t){H(t,e),"wysiwyg"===e.currentMode&&t.previousElementSibling.setAttribute("style","display:none")})),lt(e),pe(e)}else e[e.currentMode].element.focus()},rt=function(e,t,n,r){var i=(0,y.lG)(t.parentElement,"LI");if(i){e[e.currentMode].element.querySelectorAll("wbr").forEach((function(e){e.remove()})),n.insertNode(document.createElement("wbr"));var o=t.parentElement,a=o.cloneNode(),l=[t];Array.from(n.cloneContents().children).forEach((function(e,n){3!==e.nodeType&&t&&""!==e.textContent.trim()&&t.getAttribute("data-node-id")===e.getAttribute("data-node-id")&&(0!==n&&l.push(t),t=t.nextElementSibling)}));var s=!1,d="";o.querySelectorAll("li").forEach((function(e){s&&(d+=e.outerHTML,e.nextElementSibling||e.previousElementSibling?e.remove():e.parentElement.remove()),e.isSameNode(l[l.length-1])&&(s=!0)})),l.reverse().forEach((function(e){i.insertAdjacentElement("afterend",e)})),d&&(a.innerHTML=d,l[0].insertAdjacentElement("beforeend",a)),"wysiwyg"===e.currentMode?r.outerHTML=e.lute.SpinVditorDOM(r.outerHTML):r.outerHTML=e.lute.SpinVditorIRDOM(r.outerHTML),(0,N.ib)(e[e.currentMode].element,n);var c=(0,y.O9)(n.startContainer);c&&c.querySelectorAll(".vditor-"+e.currentMode+"__preview[data-render='2']").forEach((function(t){H(t,e),"wysiwyg"===e.currentMode&&t.previousElementSibling.setAttribute("style","display:none")})),lt(e),pe(e)}else e[e.currentMode].element.focus()},it=function(e,t){for(var n=getSelection().getRangeAt(0).startContainer.parentElement,r=e.rows[0].cells.length,i=e.rows.length,o=0,a=0;a<i;a++)for(var l=0;l<r;l++)if(e.rows[a].cells[l].isSameNode(n)){o=l;break}for(var s=0;s<i;s++)e.rows[s].cells[o].setAttribute("align",t)},ot=function(e){var t=e.trimRight().split("\n").pop();return""!==t&&((""===t.replace(/ |-/g,"")||""===t.replace(/ |_/g,"")||""===t.replace(/ |\*/g,""))&&(t.replace(/ /g,"").length>2&&(!(t.indexOf("-")>-1&&-1===t.trimLeft().indexOf(" ")&&e.trimRight().split("\n").length>1)&&(0!==t.indexOf("    ")&&0!==t.indexOf("\t")))))},at=function(e){var t=e.trimRight().split("\n");return 0!==(e=t.pop()).indexOf("    ")&&0!==e.indexOf("\t")&&(""!==(e=e.trimLeft())&&0!==t.length&&(""===e.replace(/-/g,"")||""===e.replace(/=/g,"")))},lt=function(e,t){void 0===t&&(t={enableAddUndoStack:!0,enableHint:!1,enableInput:!0}),"wysiwyg"===e.currentMode?X(e,t):"ir"===e.currentMode?Lt(e,t):"sv"===e.currentMode&&De(e,t)},st=function(e,t,n,r){var o,a=e.startContainer,l=(0,y.lG)(a,"LI");if(l){if(!(0,d.yl)(r)&&!r.altKey&&"Enter"===r.key&&!r.shiftKey&&n&&l.contains(n)&&n.nextElementSibling)return l&&!l.textContent.endsWith("\n")&&l.insertAdjacentText("beforeend","\n"),e.insertNode(document.createTextNode("\n\n")),e.collapse(!1),lt(t),r.preventDefault(),!0;if(!((0,d.yl)(r)||r.shiftKey||r.altKey||"Backspace"!==r.key||l.previousElementSibling||""!==e.toString()||0!==(0,N.im)(l,t[t.currentMode].element,e).start))return l.nextElementSibling?(l.parentElement.insertAdjacentHTML("beforebegin",'<p data-block="0"><wbr>'+l.innerHTML+"</p>"),l.remove()):l.parentElement.outerHTML='<p data-block="0"><wbr>'+l.innerHTML+"</p>",(0,N.ib)(t[t.currentMode].element,e),lt(t),r.preventDefault(),!0;if(!(0,d.yl)(r)&&!r.shiftKey&&!r.altKey&&"Backspace"===r.key&&""===l.textContent.trim().replace(i.g.ZWSP,"")&&""===e.toString()&&"LI"===(null===(o=l.previousElementSibling)||void 0===o?void 0:o.tagName))return l.previousElementSibling.insertAdjacentText("beforeend","\n\n"),e.selectNodeContents(l.previousElementSibling),e.collapse(!1),l.remove(),(0,N.ib)(t[t.currentMode].element,e),lt(t),r.preventDefault(),!0;if(!(0,d.yl)(r)&&!r.altKey&&"Tab"===r.key){var s=!1;if((0===e.startOffset&&(3===a.nodeType&&!a.previousSibling||3!==a.nodeType&&"LI"===a.nodeName)||l.classList.contains("vditor-task")&&1===e.startOffset&&3!==a.previousSibling.nodeType&&"INPUT"===a.previousSibling.tagName)&&(s=!0),s||""!==e.toString())return r.shiftKey?rt(t,l,e,l.parentElement):nt(t,l,e),r.preventDefault(),!0}}return!1},dt=function(e,t,n){if(e.options.tab&&"Tab"===n.key)return n.shiftKey||(""===t.toString()?(t.insertNode(document.createTextNode(e.options.tab)),t.collapse(!1)):(t.extractContents(),t.insertNode(document.createTextNode(e.options.tab)),t.collapse(!1))),(0,N.Hc)(t),lt(e),n.preventDefault(),!0},ct=function(e,t,n,r){if(n){if(!(0,d.yl)(e)&&!e.altKey&&"Enter"===e.key){var i=String.raw(F||(F=We(["",""],["",""])),n.textContent).replace(/\\\|/g,"").trim(),o=i.split("|");if(i.startsWith("|")&&i.endsWith("|")&&o.length>3){var a=o.map((function(){return"---"})).join("|");return a=n.textContent+"\n"+a.substring(3,a.length-3)+"\n|<wbr>",n.outerHTML=t.lute.SpinVditorDOM(a),(0,N.ib)(t[t.currentMode].element,r),lt(t),Te(t),e.preventDefault(),!0}if(ot(n.innerHTML)&&n.previousElementSibling){var l="",s=n.innerHTML.trimRight().split("\n");return s.length>1&&(s.pop(),l='<p data-block="0">'+s.join("\n")+"</p>"),n.insertAdjacentHTML("afterend",l+'<hr data-block="0"><p data-block="0"><wbr>\n</p>'),n.remove(),(0,N.ib)(t[t.currentMode].element,r),lt(t),Te(t),e.preventDefault(),!0}if(at(n.innerHTML))return"wysiwyg"===t.currentMode?n.outerHTML=t.lute.SpinVditorDOM(n.innerHTML+'<p data-block="0"><wbr>\n</p>'):n.outerHTML=t.lute.SpinVditorIRDOM(n.innerHTML+'<p data-block="0"><wbr>\n</p>'),(0,N.ib)(t[t.currentMode].element,r),lt(t),Te(t),e.preventDefault(),!0}if(r.collapsed&&n.previousElementSibling&&"Backspace"===e.key&&!(0,d.yl)(e)&&!e.altKey&&!e.shiftKey&&n.textContent.trimRight().split("\n").length>1&&0===(0,N.im)(n,t[t.currentMode].element,r).start){var c=(0,y.DX)(n.previousElementSibling);return c.textContent.endsWith("\n")||(c.textContent=c.textContent+"\n"),c.parentElement.insertAdjacentHTML("beforeend","<wbr>"+n.innerHTML),n.remove(),(0,N.ib)(t[t.currentMode].element,r),!1}return!1}},ut=function(e,t,n){for(var r="",i=0;i<n.parentElement.childElementCount;i++)r+='<td align="'+n.parentElement.children[i].getAttribute("align")+'"> </td>';"TH"===n.tagName?n.parentElement.parentElement.insertAdjacentHTML("afterend","<tbody><tr>"+r+"</tr></tbody>"):n.parentElement.insertAdjacentHTML("afterend","<tr>"+r+"</tr>"),lt(e)},pt=function(e,t,n){for(var r="",i=0;i<n.parentElement.childElementCount;i++)"TH"===n.tagName?r+='<th align="'+n.parentElement.children[i].getAttribute("align")+'"> </th>':r+='<td align="'+n.parentElement.children[i].getAttribute("align")+'"> </td>';if("TH"===n.tagName){n.parentElement.parentElement.insertAdjacentHTML("beforebegin","<thead><tr>"+r+"</tr></thead>"),t.insertNode(document.createElement("wbr"));var o=n.parentElement.innerHTML.replace(/<th>/g,"<td>").replace(/<\/th>/g,"</td>");n.parentElement.parentElement.nextElementSibling.insertAdjacentHTML("afterbegin",o),n.parentElement.parentElement.remove(),(0,N.ib)(e.ir.element,t)}else n.parentElement.insertAdjacentHTML("beforebegin","<tr>"+r+"</tr>");lt(e)},mt=function(e,t,n,r){void 0===r&&(r="afterend");for(var i=0,o=n.previousElementSibling;o;)i++,o=o.previousElementSibling;for(var a=0;a<t.rows.length;a++)0===a?t.rows[a].cells[i].insertAdjacentHTML(r,"<th> </th>"):t.rows[a].cells[i].insertAdjacentHTML(r,"<td> </td>");lt(e)},ft=function(e,t,n){if("TD"===n.tagName){var r=n.parentElement.parentElement;n.parentElement.previousElementSibling?t.selectNodeContents(n.parentElement.previousElementSibling.lastElementChild):t.selectNodeContents(r.previousElementSibling.lastElementChild.lastElementChild),1===r.childElementCount?r.remove():n.parentElement.remove(),t.collapse(!1),(0,N.Hc)(t),lt(e)}},ht=function(e,t,n,r){for(var i=0,o=r.previousElementSibling;o;)i++,o=o.previousElementSibling;(r.previousElementSibling||r.nextElementSibling)&&(t.selectNodeContents(r.previousElementSibling||r.nextElementSibling),t.collapse(!0));for(var a=0;a<n.rows.length;a++){var l=n.rows[a].cells;if(1===l.length){n.remove(),pe(e);break}l[i].remove()}(0,N.Hc)(t),lt(e)},vt=function(e,t,n){var r=n.startContainer,i=(0,y.lG)(r,"TD")||(0,y.lG)(r,"TH");if(i){if(!(0,d.yl)(t)&&!t.altKey&&"Enter"===t.key){i.lastElementChild&&(!i.lastElementChild||i.lastElementChild.isSameNode(i.lastChild)&&"BR"===i.lastElementChild.tagName)||i.insertAdjacentHTML("beforeend","<br>");var o=document.createElement("br");return n.insertNode(o),n.setStartAfter(o),lt(e),Te(e),t.preventDefault(),!0}if("Tab"===t.key)return t.shiftKey?(Qe(i,n),t.preventDefault(),!0):((u=i.nextElementSibling)||(u=i.parentElement.nextElementSibling?i.parentElement.nextElementSibling.firstElementChild:"THEAD"===i.parentElement.parentElement.tagName&&i.parentElement.parentElement.nextElementSibling?i.parentElement.parentElement.nextElementSibling.firstElementChild.firstElementChild:null),u&&(n.selectNodeContents(u),(0,N.Hc)(n)),t.preventDefault(),!0);var a=i.parentElement.parentElement.parentElement;if("ArrowUp"===t.key){if(t.preventDefault(),"TH"===i.tagName)return a.previousElementSibling?(n.selectNodeContents(a.previousElementSibling),n.collapse(!1),(0,N.Hc)(n)):Je(e,"beforebegin"),!0;for(var l=0,s=i.parentElement;l<s.cells.length&&!s.cells[l].isSameNode(i);l++);var c=s.previousElementSibling;return c||(c=s.parentElement.previousElementSibling.firstChild),n.selectNodeContents(c.cells[l]),n.collapse(!1),(0,N.Hc)(n),!0}if("ArrowDown"===t.key){var u;if(t.preventDefault(),!(s=i.parentElement).nextElementSibling&&"TD"===i.tagName)return a.nextElementSibling?(n.selectNodeContents(a.nextElementSibling),n.collapse(!0),(0,N.Hc)(n)):Je(e,"afterend"),!0;for(l=0;l<s.cells.length&&!s.cells[l].isSameNode(i);l++);return(u=s.nextElementSibling)||(u=s.parentElement.nextElementSibling.firstChild),n.selectNodeContents(u.cells[l]),n.collapse(!0),(0,N.Hc)(n),!0}if("wysiwyg"===e.currentMode&&!(0,d.yl)(t)&&"Enter"===t.key&&!t.shiftKey&&t.altKey){var p=e.wysiwyg.popover.querySelector(".vditor-input");return p.focus(),p.select(),t.preventDefault(),!0}if(!(0,d.yl)(t)&&!t.shiftKey&&!t.altKey&&"Backspace"===t.key&&0===n.startOffset&&""===n.toString())return!Qe(i,n,!1)&&a&&(""===a.textContent.trim()?(a.outerHTML='<p data-block="0"><wbr>\n</p>',(0,N.ib)(e[e.currentMode].element,n)):(n.setStartBefore(a),n.collapse(!0)),lt(e)),t.preventDefault(),!0;if(R("⇧⌘F",t))return pt(e,n,i),t.preventDefault(),!0;if(R("⌘=",t))return ut(e,n,i),t.preventDefault(),!0;if(R("⇧⌘G",t))return mt(e,a,i,"beforebegin"),t.preventDefault(),!0;if(R("⇧⌘=",t))return mt(e,a,i),t.preventDefault(),!0;if(R("⌘-",t))return ft(e,n,i),t.preventDefault(),!0;if(R("⇧⌘-",t))return ht(e,n,a,i),t.preventDefault(),!0;if(R("⇧⌘L",t)){if("ir"===e.currentMode)return it(a,"left"),lt(e),t.preventDefault(),!0;if(m=e.wysiwyg.popover.querySelector('[data-type="left"]'))return m.click(),t.preventDefault(),!0}if(R("⇧⌘C",t)){if("ir"===e.currentMode)return it(a,"center"),lt(e),t.preventDefault(),!0;if(m=e.wysiwyg.popover.querySelector('[data-type="center"]'))return m.click(),t.preventDefault(),!0}if(R("⇧⌘R",t)){if("ir"===e.currentMode)return it(a,"right"),lt(e),t.preventDefault(),!0;var m;if(m=e.wysiwyg.popover.querySelector('[data-type="right"]'))return m.click(),t.preventDefault(),!0}}return!1},gt=function(e,t,n,r){if("PRE"===n.tagName&&R("⌘A",t))return r.selectNodeContents(n.firstElementChild),t.preventDefault(),!0;if(e.options.tab&&"Tab"===t.key&&!t.shiftKey&&""===r.toString())return r.insertNode(document.createTextNode(e.options.tab)),r.collapse(!1),lt(e),t.preventDefault(),!0;if("Backspace"===t.key&&!(0,d.yl)(t)&&!t.shiftKey&&!t.altKey){var i=(0,N.im)(n,e[e.currentMode].element,r);if((0===i.start||1===i.start&&"\n"===n.innerText)&&""===r.toString())return n.parentElement.outerHTML='<p data-block="0"><wbr>'+n.firstElementChild.innerHTML+"</p>",(0,N.ib)(e[e.currentMode].element,r),lt(e),t.preventDefault(),!0}return!(0,d.yl)(t)&&!t.altKey&&"Enter"===t.key&&(n.firstElementChild.textContent.endsWith("\n")||n.firstElementChild.insertAdjacentText("beforeend","\n"),r.extractContents(),r.insertNode(document.createTextNode("\n")),r.collapse(!1),(0,N.Hc)(r),(0,d.vU)()||("wysiwyg"===e.currentMode?Ue(e,r):j(e,r)),Te(e),t.preventDefault(),!0)},yt=function(e,t,n,r){var o=t.startContainer,a=(0,y.lG)(o,"BLOCKQUOTE");if(a&&""===t.toString()){if("Backspace"===n.key&&!(0,d.yl)(n)&&!n.shiftKey&&!n.altKey&&0===(0,N.im)(a,e[e.currentMode].element,t).start)return t.insertNode(document.createElement("wbr")),a.outerHTML=a.innerHTML,(0,N.ib)(e[e.currentMode].element,t),lt(e),n.preventDefault(),!0;if(r&&"Enter"===n.key&&!(0,d.yl)(n)&&!n.shiftKey&&!n.altKey&&"BLOCKQUOTE"===r.parentElement.tagName){var l=!1;if("\n"===r.innerHTML.replace(i.g.ZWSP,"")||""===r.innerHTML.replace(i.g.ZWSP,"")?(l=!0,r.remove()):r.innerHTML.endsWith("\n\n")&&(0,N.im)(r,e[e.currentMode].element,t).start===r.textContent.length-1&&(r.innerHTML=r.innerHTML.substr(0,r.innerHTML.length-2),l=!0),l)return a.insertAdjacentHTML("afterend",'<p data-block="0">'+i.g.ZWSP+"<wbr>\n</p>"),(0,N.ib)(e[e.currentMode].element,t),lt(e),n.preventDefault(),!0}var s=(0,y.F9)(o);if("wysiwyg"===e.currentMode&&s&&R("⇧⌘;",n))return t.insertNode(document.createElement("wbr")),s.outerHTML='<blockquote data-block="0">'+s.outerHTML+"</blockquote>",(0,N.ib)(e.wysiwyg.element,t),X(e),n.preventDefault(),!0;if($e(e,n,t,a,a))return!0;if(et(e,n,t,a,a))return!0}return!1},bt=function(e,t,n){var r=t.startContainer,i=(0,y.fb)(r,"vditor-task");if(i){if(R("⇧⌘J",n)){var o=i.firstElementChild;return o.checked?o.removeAttribute("checked"):o.setAttribute("checked","checked"),lt(e),n.preventDefault(),!0}if("Backspace"===n.key&&!(0,d.yl)(n)&&!n.shiftKey&&!n.altKey&&""===t.toString()&&1===t.startOffset&&(3===r.nodeType&&r.previousSibling&&"INPUT"===r.previousSibling.tagName||3!==r.nodeType)){var a=i.previousElementSibling;if(i.querySelector("input").remove(),a)(0,y.DX)(a).parentElement.insertAdjacentHTML("beforeend","<wbr>"+i.innerHTML.trim()),i.remove();else i.parentElement.insertAdjacentHTML("beforebegin",'<p data-block="0"><wbr>'+(i.innerHTML.trim()||"\n")+"</p>"),i.nextElementSibling?i.remove():i.parentElement.remove();return(0,N.ib)(e[e.currentMode].element,t),lt(e),n.preventDefault(),!0}if("Enter"===n.key&&!(0,d.yl)(n)&&!n.shiftKey&&!n.altKey){if(""===i.textContent.trim())if((0,y.fb)(i.parentElement,"vditor-task")){var l=(0,y.O9)(r);l&&rt(e,i,t,l)}else if(i.nextElementSibling){var s="",c="",u=!1;Array.from(i.parentElement.children).forEach((function(e){i.isSameNode(e)?u=!0:u?s+=e.outerHTML:c+=e.outerHTML}));var p=i.parentElement.tagName,m="OL"===i.parentElement.tagName?"":' data-marker="'+i.parentElement.getAttribute("data-marker")+'"',f="";c&&(f="UL"===i.parentElement.tagName?"":' start="1"',c="<"+p+' data-tight="true"'+m+' data-block="0">'+c+"</"+p+">"),i.parentElement.outerHTML=c+'<p data-block="0"><wbr>\n</p><'+p+'\n data-tight="true"'+m+' data-block="0"'+f+">"+s+"</"+p+">"}else i.parentElement.insertAdjacentHTML("afterend",'<p data-block="0"><wbr>\n</p>'),1===i.parentElement.querySelectorAll("li").length?i.parentElement.remove():i.remove();else 3!==r.nodeType&&0===t.startOffset&&"INPUT"===r.firstChild.tagName?t.setStart(r.childNodes[1],1):(t.setEndAfter(i.lastChild),i.insertAdjacentHTML("afterend",'<li class="vditor-task" data-marker="'+i.getAttribute("data-marker")+'"><input type="checkbox"> <wbr></li>'),document.querySelector("wbr").after(t.extractContents()));return(0,N.ib)(e[e.currentMode].element,t),lt(e),Te(e),n.preventDefault(),!0}}return!1},wt=function(e,t,n,r){if(3!==t.startContainer.nodeType){var i=t.startContainer.children[t.startOffset];if(i&&"HR"===i.tagName)return t.selectNodeContents(i.previousElementSibling),t.collapse(!1),n.preventDefault(),!0}if(r){var o=r.previousElementSibling;if(o&&0===(0,N.im)(r,e[e.currentMode].element,t).start&&((0,d.vU)()&&"HR"===o.tagName||"TABLE"===o.tagName)){if("TABLE"===o.tagName){var a=o.lastElementChild.lastElementChild.lastElementChild;a.innerHTML=a.innerHTML.trimLeft()+"<wbr>"+r.textContent.trim(),r.remove()}else o.remove();return(0,N.ib)(e[e.currentMode].element,t),lt(e),n.preventDefault(),!0}}return!1},Et=function(e){(0,d.vU)()&&3!==e.startContainer.nodeType&&"HR"===e.startContainer.tagName&&e.setStartBefore(e.startContainer)},kt=function(e,t,n){var r,i;if(!(0,d.vU)())return!1;if("ArrowUp"===e.key&&t&&"TABLE"===(null===(r=t.previousElementSibling)||void 0===r?void 0:r.tagName)){var o=t.previousElementSibling;return n.selectNodeContents(o.rows[o.rows.length-1].lastElementChild),n.collapse(!1),e.preventDefault(),!0}return!("ArrowDown"!==e.key||!t||"TABLE"!==(null===(i=t.nextElementSibling)||void 0===i?void 0:i.tagName))&&(n.selectNodeContents(t.nextElementSibling.rows[0].cells[0]),n.collapse(!0),e.preventDefault(),!0)},St=function(e,t,n){return ze(void 0,void 0,void 0,(function(){var r,o,a,l,s,d,c,u,p,m,f,h,v,g;return Ge(this,(function(b){switch(b.label){case 0:return t.stopPropagation(),t.preventDefault(),"clipboardData"in t?(r=t.clipboardData.getData("text/html"),o=t.clipboardData.getData("text/plain"),a=t.clipboardData.files):(r=t.dataTransfer.getData("text/html"),o=t.dataTransfer.getData("text/plain"),t.dataTransfer.types.includes("Files")&&(a=t.dataTransfer.items)),l={},s=function(t,n){if(!n)return["",Lute.WalkContinue];var r=t.TokensStr();if(34===t.__internal_object__.Parent.Type&&r&&-1===r.indexOf("file://")&&e.options.upload.linkToImgUrl){var i=new XMLHttpRequest;i.open("POST",e.options.upload.linkToImgUrl),e.options.upload.token&&i.setRequestHeader("X-Upload-Token",e.options.upload.token),e.options.upload.withCredentials&&(i.withCredentials=!0),Re(e,i),i.setRequestHeader("Content-Type","application/json; charset=utf-8"),i.onreadystatechange=function(){if(i.readyState===XMLHttpRequest.DONE){if(200===i.status){var t=i.responseText;e.options.upload.linkToImgFormat&&(t=e.options.upload.linkToImgFormat(i.responseText));var n=JSON.parse(t);if(0!==n.code)return void e.tip.show(n.msg);var r=n.data.originalURL;if("sv"===e.currentMode)e.sv.element.querySelectorAll(".vditor-sv__marker--link").forEach((function(e){e.textContent===r&&(e.textContent=n.data.url)}));else{var o=e[e.currentMode].element.querySelector('img[src="'+r+'"]');o.src=n.data.url,"ir"===e.currentMode&&(o.previousElementSibling.previousElementSibling.innerHTML=n.data.url)}lt(e)}else e.tip.show(i.responseText);e.options.upload.linkToImgCallback&&e.options.upload.linkToImgCallback(i.responseText)}},i.send(JSON.stringify({url:r}))}return"ir"===e.currentMode?['<span class="vditor-ir__marker vditor-ir__marker--link">'+r+"</span>",Lute.WalkContinue]:"wysiwyg"===e.currentMode?["",Lute.WalkContinue]:['<span class="vditor-sv__marker--link">'+r+"</span>",Lute.WalkContinue]},r.replace(/&amp;/g,"&").replace(/<(|\/)(html|body|meta)[^>]*?>/gi,"").trim()!=='<a href="'+o+'">'+o+"</a>"&&r.replace(/&amp;/g,"&").replace(/<(|\/)(html|body|meta)[^>]*?>/gi,"").trim()!=='\x3c!--StartFragment--\x3e<a href="'+o+'">'+o+"</a>\x3c!--EndFragment--\x3e"||(r=""),(d=(new DOMParser).parseFromString(r,"text/html")).body&&(r=d.body.innerHTML),r=Lute.Sanitize(r),e.wysiwyg.getComments(e),c=e[e.currentMode].element.scrollHeight,u=function(e,t,n){void 0===n&&(n="sv");var r=document.createElement("div");r.innerHTML=e;var i=!1;1===r.childElementCount&&r.lastElementChild.style.fontFamily.indexOf("monospace")>-1&&(i=!0);var o=r.querySelectorAll("pre");if(1===r.childElementCount&&1===o.length&&"vditor-wysiwyg"!==o[0].className&&"vditor-sv"!==o[0].className&&(i=!0),0===e.indexOf('\n<p class="p1">')&&(i=!0),1===r.childElementCount&&"TABLE"===r.firstElementChild.tagName&&r.querySelector(".line-number")&&r.querySelector(".line-content")&&(i=!0),i){var a=t||e;return/\n/.test(a)||1===o.length?"wysiwyg"===n?'<div class="vditor-wysiwyg__block" data-block="0" data-type="code-block"><pre><code>'+a.replace(/&/g,"&amp;").replace(/</g,"&lt;")+"<wbr></code></pre></div>":"\n```\n"+a.replace(/&/g,"&amp;").replace(/</g,"&lt;")+"\n```":"wysiwyg"===n?"<code>"+a.replace(/&/g,"&amp;").replace(/</g,"&lt;")+"</code><wbr>":"`"+a+"`"}return!1}(r,o,e.currentMode),(p="sv"===e.currentMode?(0,y.a1)(t.target,"data-type","code-block"):(0,y.lG)(t.target,"CODE"))?("sv"===e.currentMode?document.execCommand("insertHTML",!1,o.replace(/&/g,"&amp;").replace(/</g,"&lt;")):(m=(0,N.im)(t.target,e[e.currentMode].element),"PRE"!==p.parentElement.tagName&&(o+=i.g.ZWSP),p.textContent=p.textContent.substring(0,m.start)+o+p.textContent.substring(m.end),(0,N.$j)(m.start+o.length,m.start+o.length,p.parentElement),(null===(g=p.parentElement)||void 0===g?void 0:g.nextElementSibling.classList.contains("vditor-"+e.currentMode+"__preview"))&&(p.parentElement.nextElementSibling.innerHTML=p.outerHTML,H(p.parentElement.nextElementSibling,e))),[3,6]):[3,1];case 1:return u?(n.pasteCode(u),[3,6]):[3,2];case 2:return""===r.trim()?[3,3]:((f=document.createElement("div")).innerHTML=r,f.querySelectorAll("[style]").forEach((function(e){e.removeAttribute("style")})),f.querySelectorAll(".vditor-copy").forEach((function(e){e.remove()})),"ir"===e.currentMode?(l.HTML2VditorIRDOM={renderLinkDest:s},e.lute.SetJSRenderers({renderers:l}),(0,N.oC)(e.lute.HTML2VditorIRDOM(f.innerHTML),e)):"wysiwyg"===e.currentMode?(l.HTML2VditorDOM={renderLinkDest:s},e.lute.SetJSRenderers({renderers:l}),(0,N.oC)(e.lute.HTML2VditorDOM(f.innerHTML),e)):(l.Md2VditorSVDOM={renderLinkDest:s},e.lute.SetJSRenderers({renderers:l}),_e(e,e.lute.HTML2Md(f.innerHTML).trimRight())),e.outline.render(e),[3,6]);case 3:return a.length>0&&(e.options.upload.url||e.options.upload.handler)?[4,Ve(e,a)]:[3,5];case 4:return b.sent(),[3,6];case 5:""!==o.trim()&&0===a.length&&("ir"===e.currentMode?(l.Md2VditorIRDOM={renderLinkDest:s},e.lute.SetJSRenderers({renderers:l}),(0,N.oC)(e.lute.Md2VditorIRDOM(o),e)):"wysiwyg"===e.currentMode?(l.Md2VditorDOM={renderLinkDest:s},e.lute.SetJSRenderers({renderers:l}),(0,N.oC)(e.lute.Md2VditorDOM(o),e)):(l.Md2VditorSVDOM={renderLinkDest:s},e.lute.SetJSRenderers({renderers:l}),_e(e,o)),e.outline.render(e)),b.label=6;case 6:return"sv"!==e.currentMode&&((h=(0,y.F9)((0,N.zh)(e).startContainer))&&(v=(0,N.zh)(e),e[e.currentMode].element.querySelectorAll("wbr").forEach((function(e){e.remove()})),v.insertNode(document.createElement("wbr")),"wysiwyg"===e.currentMode?h.outerHTML=e.lute.SpinVditorDOM(h.outerHTML):h.outerHTML=e.lute.SpinVditorIRDOM(h.outerHTML),(0,N.ib)(e[e.currentMode].element,v)),e[e.currentMode].element.querySelectorAll(".vditor-"+e.currentMode+"__preview[data-render='2']").forEach((function(t){H(t,e)}))),e.wysiwyg.triggerRemoveComment(e),lt(e),e[e.currentMode].element.scrollHeight-c>Math.min(e[e.currentMode].element.clientHeight,window.innerHeight)/2&&Te(e),[2]}}))}))},Ct=function(e){e.hint.render(e);var t=(0,N.zh)(e).startContainer,n=(0,y.a1)(t,"data-type","code-block-info");if(n)if(""===n.textContent.replace(i.g.ZWSP,"")&&e.hint.recentLanguage){n.textContent=i.g.ZWSP+e.hint.recentLanguage,(0,N.zh)(e).selectNodeContents(n)}else{var r=[],o=n.textContent.substring(0,(0,N.im)(n,e.ir.element).start).replace(i.g.ZWSP,"");i.g.CODE_LANGUAGES.forEach((function(e){e.indexOf(o.toLowerCase())>-1&&r.push({html:e,value:e})})),e.hint.genHTML(r,o,e)}},Lt=function(e,t){void 0===t&&(t={enableAddUndoStack:!0,enableHint:!1,enableInput:!0}),t.enableHint&&Ct(e),clearTimeout(e.ir.processTimeoutId),e.ir.processTimeoutId=window.setTimeout((function(){if(!e.ir.composingLock){var n=a(e);"function"==typeof e.options.input&&t.enableInput&&e.options.input(n),e.options.counter.enable&&e.counter.render(e,n),e.options.cache.enable&&(0,d.pK)()&&(localStorage.setItem(e.options.cache.id,n),e.options.cache.after&&e.options.cache.after(n)),e.devtools&&e.devtools.renderEchart(e),t.enableAddUndoStack&&e.undo.addToUndoStack(e)}}),e.options.undoDelay)},Tt=function(e,t){var n=(0,N.zh)(e),r=(0,y.F9)(n.startContainer)||n.startContainer;if(r){var i=r.querySelector(".vditor-ir__marker--heading");i?i.innerHTML=t:(r.insertAdjacentText("afterbegin",t),n.selectNodeContents(r),n.collapse(!1)),j(e,n.cloneRange()),J(e)}},Mt=function(e,t,n){var r=(0,y.a1)(e.startContainer,"data-type",n);if(r){r.firstElementChild.remove(),r.lastElementChild.remove(),e.insertNode(document.createElement("wbr"));var i=document.createElement("div");i.innerHTML=t.lute.SpinVditorIRDOM(r.outerHTML),r.outerHTML=i.firstElementChild.innerHTML.trim()}},At=function(e,t,n,r){var i=(0,N.zh)(e),o=t.getAttribute("data-type"),a=i.startContainer;3===a.nodeType&&(a=a.parentElement);var l=!0;if(t.classList.contains("vditor-menu--current"))if("quote"===o){var s=(0,y.lG)(a,"BLOCKQUOTE");s&&(i.insertNode(document.createElement("wbr")),s.outerHTML=""===s.innerHTML.trim()?'<p data-block="0">'+s.innerHTML+"</p>":s.innerHTML)}else if("link"===o){var d=(0,y.a1)(i.startContainer,"data-type","a");if(d){var u=(0,y.fb)(i.startContainer,"vditor-ir__link");u?(i.insertNode(document.createElement("wbr")),d.outerHTML=u.innerHTML):d.outerHTML=d.querySelector(".vditor-ir__link").innerHTML+"<wbr>"}}else"italic"===o?Mt(i,e,"em"):"bold"===o?Mt(i,e,"strong"):"strike"===o?Mt(i,e,"s"):"inline-code"===o?Mt(i,e,"code"):"check"!==o&&"list"!==o&&"ordered-list"!==o||(tt(e,i,o),l=!1,t.classList.remove("vditor-menu--current"));else{0===e.ir.element.childNodes.length&&(e.ir.element.innerHTML='<p data-block="0"><wbr></p>',(0,N.ib)(e.ir.element,i));var p=(0,y.F9)(i.startContainer);if("line"===o){if(p){var m='<hr data-block="0"><p data-block="0"><wbr>\n</p>';""===p.innerHTML.trim()?p.outerHTML=m:p.insertAdjacentHTML("afterend",m)}}else if("quote"===o)p&&(i.insertNode(document.createElement("wbr")),p.outerHTML='<blockquote data-block="0">'+p.outerHTML+"</blockquote>",l=!1,t.classList.add("vditor-menu--current"));else if("link"===o){var f=void 0;f=""===i.toString()?n+"<wbr>"+r:""+n+i.toString()+r.replace(")","<wbr>)"),document.execCommand("insertHTML",!1,f),l=!1,t.classList.add("vditor-menu--current")}else if("italic"===o||"bold"===o||"strike"===o||"inline-code"===o||"code"===o||"table"===o){f=void 0;""===i.toString()?f=n+"<wbr>"+r:(f="code"===o||"table"===o?""+n+i.toString()+"<wbr>"+r:""+n+i.toString()+r+"<wbr>",i.deleteContents()),"table"!==o&&"code"!==o||(f="\n"+f+"\n\n");var h=document.createElement("span");h.innerHTML=f,i.insertNode(h),j(e,i),"table"===o&&(i.selectNodeContents(getSelection().getRangeAt(0).startContainer.parentElement),(0,N.Hc)(i))}else"check"!==o&&"list"!==o&&"ordered-list"!==o||(tt(e,i,o,!1),l=!1,c(e.toolbar.elements,["check","list","ordered-list"]),t.classList.add("vditor-menu--current"))}(0,N.ib)(e.ir.element,i),Lt(e),l&&J(e)},_t=function(e,t,n,r){return new(n||(n=Promise))((function(i,o){function a(e){try{s(r.next(e))}catch(e){o(e)}}function l(e){try{s(r.throw(e))}catch(e){o(e)}}function s(e){var t;e.done?i(e.value):(t=e.value,t instanceof n?t:new n((function(e){e(t)}))).then(a,l)}s((r=r.apply(e,t||[])).next())}))},xt=function(e,t){var n,r,i,o,a={label:0,sent:function(){if(1&i[0])throw i[1];return i[1]},trys:[],ops:[]};return o={next:l(0),throw:l(1),return:l(2)},"function"==typeof Symbol&&(o[Symbol.iterator]=function(){return this}),o;function l(o){return function(l){return function(o){if(n)throw new TypeError("Generator is already executing.");for(;a;)try{if(n=1,r&&(i=2&o[0]?r.return:o[0]?r.throw||((i=r.return)&&i.call(r),0):r.next)&&!(i=i.call(r,o[1])).done)return i;switch(r=0,i&&(o=[2&o[0],i.value]),o[0]){case 0:case 1:i=o;break;case 4:return a.label++,{value:o[1],done:!1};case 5:a.label++,r=o[1],o=[0];continue;case 7:o=a.ops.pop(),a.trys.pop();continue;default:if(!(i=a.trys,(i=i.length>0&&i[i.length-1])||6!==o[0]&&2!==o[0])){a=0;continue}if(3===o[0]&&(!i||o[1]>i[0]&&o[1]<i[3])){a.label=o[1];break}if(6===o[0]&&a.label<i[1]){a.label=i[1],i=o;break}if(i&&a.label<i[2]){a.label=i[2],a.ops.push(o);break}i[2]&&a.ops.pop(),a.trys.pop();continue}o=t.call(e,a)}catch(e){o=[6,e],r=0}finally{n=i=0}if(5&o[0])throw o[1];return{value:o[0]?o[1]:void 0,done:!0}}([o,l])}}},Ht=function(){function e(e){var t=this;this.splitChar="",this.lastIndex=-1,this.fillEmoji=function(e,n){t.element.style.display="none";var r=decodeURIComponent(e.getAttribute("data-value")),o=window.getSelection().getRangeAt(0);if("ir"===n.currentMode){var a=(0,y.a1)(o.startContainer,"data-type","code-block-info");if(a)return a.textContent=i.g.ZWSP+r.trimRight(),o.selectNodeContents(a),o.collapse(!1),Lt(n),a.parentElement.querySelectorAll("code").forEach((function(e){e.className="language-"+r.trimRight()})),H(a.parentElement.querySelector(".vditor-ir__preview"),n),void(t.recentLanguage=r.trimRight())}if("wysiwyg"===n.currentMode&&3!==o.startContainer.nodeType){var l=o.startContainer,s=void 0;if((s=l.classList.contains("vditor-input")?l:l.firstElementChild)&&s.classList.contains("vditor-input"))return s.value=r.trimRight(),o.selectNodeContents(s),o.collapse(!1),s.dispatchEvent(new CustomEvent("input",{detail:1})),void(t.recentLanguage=r.trimRight())}if(o.setStart(o.startContainer,t.lastIndex),o.deleteContents(),n.options.hint.parse?"sv"===n.currentMode?(0,N.oC)(n.lute.SpinVditorSVDOM(r),n):"wysiwyg"===n.currentMode?(0,N.oC)(n.lute.SpinVditorDOM(r),n):(0,N.oC)(n.lute.SpinVditorIRDOM(r),n):(0,N.oC)(r,n),":"===t.splitChar&&r.indexOf(":")>-1&&"sv"!==n.currentMode&&o.insertNode(document.createTextNode(" ")),o.collapse(!1),(0,N.Hc)(o),"wysiwyg"===n.currentMode)(d=(0,y.fb)(o.startContainer,"vditor-wysiwyg__block"))&&d.lastElementChild.classList.contains("vditor-wysiwyg__preview")&&(d.lastElementChild.innerHTML=d.firstElementChild.innerHTML,H(d.lastElementChild,n));else if("ir"===n.currentMode){var d;(d=(0,y.fb)(o.startContainer,"vditor-ir__marker--pre"))&&d.nextElementSibling.classList.contains("vditor-ir__preview")&&(d.nextElementSibling.innerHTML=d.innerHTML,H(d.nextElementSibling,n))}lt(n)},this.timeId=-1,this.element=document.createElement("div"),this.element.className="vditor-hint",this.recentLanguage="",e.push({key:":"})}return e.prototype.render=function(e){var t=this;if(window.getSelection().focusNode){var n,r=getSelection().getRangeAt(0);n=r.startContainer.textContent.substring(0,r.startOffset)||"";var i=this.getKey(n,e.options.hint.extend);if(void 0===i)this.element.style.display="none",clearTimeout(this.timeId);else if(":"===this.splitChar){var o=""===i?e.options.hint.emoji:e.lute.GetEmojis(),a=[];Object.keys(o).forEach((function(e){0===e.indexOf(i.toLowerCase())&&(o[e].indexOf(".")>-1?a.push({html:'<img src="'+o[e]+'" title=":'+e+':"/> :'+e+":",value:":"+e+":"}):a.push({html:'<span class="vditor-hint__emoji">'+o[e]+"</span>"+e,value:o[e]}))})),this.genHTML(a,i,e)}else e.options.hint.extend.forEach((function(n){n.key===t.splitChar&&(clearTimeout(t.timeId),t.timeId=window.setTimeout((function(){return _t(t,void 0,void 0,(function(){var t;return xt(this,(function(r){switch(r.label){case 0:return t=this.genHTML,[4,n.hint(i)];case 1:return t.apply(this,[r.sent(),i,e]),[2]}}))}))}),e.options.hint.delay))}))}},e.prototype.genHTML=function(e,t,n){var r=this;if(0!==e.length){var i=n[n.currentMode].element,o=(0,N.Ny)(i),a=o.left+("left"===n.options.outline.position?n.outline.element.offsetWidth:0),l=o.top,s="";e.forEach((function(e,n){if(!(n>7)){var r=e.html;if(""!==t){var i=r.lastIndexOf(">")+1,o=r.substr(i),a=o.toLowerCase().indexOf(t.toLowerCase());a>-1&&(o=o.substring(0,a)+"<b>"+o.substring(a,a+t.length)+"</b>"+o.substring(a+t.length),r=r.substr(0,i)+o)}s+='<button data-value="'+encodeURIComponent(e.value)+' "\n'+(0===n?"class='vditor-hint--current'":"")+"> "+r+"</button>"}})),this.element.innerHTML=s;var d=parseInt(document.defaultView.getComputedStyle(i,null).getPropertyValue("line-height"),10);this.element.style.top=l+(d||22)+"px",this.element.style.left=a+"px",this.element.style.display="block",this.element.style.right="auto",this.element.querySelectorAll("button").forEach((function(e){e.addEventListener("click",(function(t){r.fillEmoji(e,n),t.preventDefault()}))})),this.element.getBoundingClientRect().bottom>window.innerHeight&&(this.element.style.top=l-this.element.offsetHeight+"px"),this.element.getBoundingClientRect().right>window.innerWidth&&(this.element.style.left="auto",this.element.style.right="0")}else this.element.style.display="none"},e.prototype.select=function(e,t){if(0===this.element.querySelectorAll("button").length||"none"===this.element.style.display)return!1;var n=this.element.querySelector(".vditor-hint--current");if("ArrowDown"===e.key)return e.preventDefault(),e.stopPropagation(),n.removeAttribute("class"),n.nextElementSibling?n.nextElementSibling.className="vditor-hint--current":this.element.children[0].className="vditor-hint--current",!0;if("ArrowUp"===e.key){if(e.preventDefault(),e.stopPropagation(),n.removeAttribute("class"),n.previousElementSibling)n.previousElementSibling.className="vditor-hint--current";else{var r=this.element.children.length;this.element.children[r-1].className="vditor-hint--current"}return!0}return!((0,d.yl)(e)||e.shiftKey||e.altKey||"Enter"!==e.key||e.isComposing)&&(e.preventDefault(),e.stopPropagation(),this.fillEmoji(n,t),!0)},e.prototype.getKey=function(e,t){var n,r=this;if(this.lastIndex=-1,this.splitChar="",t.forEach((function(t){var n=e.lastIndexOf(t.key);r.lastIndex<n&&(r.splitChar=t.key,r.lastIndex=n)})),-1===this.lastIndex)return n;var i=e.split(this.splitChar),a=i[i.length-1];if(i.length>1&&a.trim()===a)if(2===i.length&&""===i[0]&&i[1].length<32)n=i[1];else{var l=i[i.length-2].slice(-1);" "===(0,o.X)(l)&&a.length<32&&(n=a)}return n},e}(),Nt=function(){function e(e){this.composingLock=!1;var t=document.createElement("div");t.className="vditor-ir",t.innerHTML='<pre class="vditor-reset" placeholder="'+e.options.placeholder+'"\n contenteditable="true" spellcheck="false"></pre>',this.element=t.firstElementChild,this.bindEvent(e),we(e,this.element),Ee(e,this.element),ke(e,this.element),Me(e,this.element),Ae(e,this.element),Se(e,this.element),Ce(e,this.element,this.copy),Le(e,this.element,this.copy)}return e.prototype.copy=function(e,t){var n=getSelection().getRangeAt(0);if(""!==n.toString()){e.stopPropagation(),e.preventDefault();var r=document.createElement("div");r.appendChild(n.cloneContents()),e.clipboardData.setData("text/plain",t.lute.VditorIRDOM2Md(r.innerHTML).trim()),e.clipboardData.setData("text/html","")}},e.prototype.bindEvent=function(e){var t=this;this.element.addEventListener("paste",(function(t){St(e,t,{pasteCode:function(e){document.execCommand("insertHTML",!1,e)}})})),this.element.addEventListener("compositionstart",(function(e){t.composingLock=!0})),this.element.addEventListener("compositionend",(function(n){(0,d.vU)()||j(e,getSelection().getRangeAt(0).cloneRange()),t.composingLock=!1})),this.element.addEventListener("input",(function(n){"deleteByDrag"!==n.inputType&&"insertFromDrop"!==n.inputType&&(t.preventInput?t.preventInput=!1:t.composingLock||"‘"===n.data||"“"===n.data||"《"===n.data||j(e,getSelection().getRangeAt(0).cloneRange(),!1,n))})),this.element.addEventListener("click",(function(n){if("INPUT"===n.target.tagName)return n.target.checked?n.target.setAttribute("checked","checked"):n.target.removeAttribute("checked"),t.preventInput=!0,void Lt(e);var r=(0,N.zh)(e),o=(0,y.fb)(n.target,"vditor-ir__preview");if(o||(o=(0,y.fb)(r.startContainer,"vditor-ir__preview")),o&&(o.previousElementSibling.firstElementChild?r.selectNodeContents(o.previousElementSibling.firstElementChild):r.selectNodeContents(o.previousElementSibling),r.collapse(!0),(0,N.Hc)(r),Te(e)),"IMG"===n.target.tagName){var a=n.target.parentElement.querySelector(".vditor-ir__marker--link");a&&(r.selectNode(a),(0,N.Hc)(r))}var l=(0,y.a1)(n.target,"data-type","a");if(!l||l.classList.contains("vditor-ir__node--expand")){if(n.target.isEqualNode(t.element)&&t.element.lastElementChild&&r.collapsed){var s=t.element.lastElementChild.getBoundingClientRect();n.y>s.top+s.height&&("P"===t.element.lastElementChild.tagName&&""===t.element.lastElementChild.textContent.trim().replace(i.g.ZWSP,"")?(r.selectNodeContents(t.element.lastElementChild),r.collapse(!1)):(t.element.insertAdjacentHTML("beforeend",'<p data-block="0">'+i.g.ZWSP+"<wbr></p>"),(0,N.ib)(t.element,r)))}""===r.toString()?P(r,e):setTimeout((function(){P((0,N.zh)(e),e)})),O(n,e),J(e)}else window.open(l.querySelector(":scope > .vditor-ir__marker--link").textContent)})),this.element.addEventListener("keyup",(function(n){if(!n.isComposing&&!(0,d.yl)(n))if("Enter"===n.key&&Te(e),J(e),"Backspace"!==n.key&&"Delete"!==n.key||""===e.ir.element.innerHTML||1!==e.ir.element.childNodes.length||!e.ir.element.firstElementChild||"P"!==e.ir.element.firstElementChild.tagName||0!==e.ir.element.firstElementChild.childElementCount||""!==e.ir.element.textContent&&"\n"!==e.ir.element.textContent){var r=(0,N.zh)(e);"Backspace"===n.key?((0,d.vU)()&&"\n"===r.startContainer.textContent&&1===r.startOffset&&(r.startContainer.textContent="",P(r,e)),t.element.querySelectorAll(".language-math").forEach((function(e){var t=e.querySelector("br");t&&t.remove()}))):n.key.indexOf("Arrow")>-1?("ArrowLeft"!==n.key&&"ArrowRight"!==n.key||Ct(e),P(r,e)):229===n.keyCode&&""===n.code&&"Unidentified"===n.key&&P(r,e);var o=(0,y.fb)(r.startContainer,"vditor-ir__preview");if(o){if("ArrowUp"===n.key||"ArrowLeft"===n.key)return o.previousElementSibling.firstElementChild?r.selectNodeContents(o.previousElementSibling.firstElementChild):r.selectNodeContents(o.previousElementSibling),r.collapse(!1),n.preventDefault(),!0;if("SPAN"===o.tagName&&("ArrowDown"===n.key||"ArrowRight"===n.key))return"html-entity"===o.parentElement.getAttribute("data-type")?(o.parentElement.insertAdjacentText("afterend",i.g.ZWSP),r.setStart(o.parentElement.nextSibling,1)):r.selectNodeContents(o.parentElement.lastElementChild),r.collapse(!1),n.preventDefault(),!0}}else e.ir.element.innerHTML=""}))},e}(),Dt=function(e){return"sv"===e.currentMode?e.lute.Md2HTML(a(e)):"wysiwyg"===e.currentMode?e.lute.VditorDOM2HTML(e.wysiwyg.element.innerHTML):"ir"===e.currentMode?e.lute.VditorIRDOM2HTML(e.ir.element.innerHTML):void 0},Ot=n(792),It=n(198),jt=function(){function e(e){this.element=document.createElement("div"),this.element.className="vditor-outline",this.element.innerHTML='<div class="vditor-outline__title">'+e+'</div>\n<div class="vditor-outline__content"></div>'}return e.prototype.render=function(e){return"block"===e.preview.element.style.display?(0,It.k)(e.preview.element.lastElementChild,this.element.lastElementChild,e):(0,It.k)(e[e.currentMode].element,this.element.lastElementChild,e)},e.prototype.toggle=function(e,t){var n;void 0===t&&(t=!0);var r=null===(n=e.toolbar.elements.outline)||void 0===n?void 0:n.firstElementChild;if(t&&window.innerWidth>=i.g.MOBILE_WIDTH?(this.element.style.display="block",this.render(e),null==r||r.classList.add("vditor-menu--current")):(this.element.style.display="none",null==r||r.classList.remove("vditor-menu--current")),getSelection().rangeCount>0){var o=getSelection().getRangeAt(0);e[e.currentMode].element.contains(o.startContainer)?(0,N.Hc)(o):e[e.currentMode].element.focus()}W(e)},e}(),Rt=n(207),Pt=function(){function e(e){var t=this;this.element=document.createElement("div"),this.element.className="vditor-preview";var n=document.createElement("div");n.className="vditor-reset",e.options.classes.preview&&n.classList.add(e.options.classes.preview),n.style.maxWidth=e.options.preview.maxWidth+"px",n.addEventListener("copy",(function(n){if("TEXTAREA"!==n.target.tagName){var r=document.createElement("div");r.className="vditor-reset",r.appendChild(getSelection().getRangeAt(0).cloneContents()),t.copyToX(e,r),n.preventDefault()}})),n.addEventListener("click",(function(r){var i=(0,y.lG)(r.target,"SPAN");if(i&&(0,y.fb)(i,"vditor-toc")){var o=n.querySelector("#"+i.getAttribute("data-target-id"));o&&(t.element.scrollTop=o.offsetTop)}else"IMG"===r.target.tagName&&(0,B.E)(r.target,e.options.lang,e.options.theme)}));var r=e.options.preview.actions,i=document.createElement("div");i.className="vditor-preview__action";for(var o=[],a=0;a<r.length;a++){var l=r[a];if("object"!=typeof l)switch(l){case"desktop":o.push('<button type="button" class="vditor-preview__action--current" data-type="desktop">Desktop</button>');break;case"tablet":o.push('<button type="button" data-type="tablet">Tablet</button>');break;case"mobile":o.push('<button type="button" data-type="mobile">Mobile/Wechat</button>');break;case"mp-wechat":o.push('<button type="button" data-type="mp-wechat" class="vditor-tooltipped vditor-tooltipped__w" aria-label="复制到公众号"><svg><use xlink:href="#vditor-icon-mp-wechat"></use></svg></button>');break;case"zhihu":o.push('<button type="button" data-type="zhihu" class="vditor-tooltipped vditor-tooltipped__w" aria-label="复制到知乎"><svg><use xlink:href="#vditor-icon-zhihu"></use></svg></button>')}else o.push('<button type="button" data-type="'+l.key+'" class="'+l.className+'"'+(l.tooltip?' aria-label="'+l.tooltip+'"':"")+'">'+l.text+"</button>")}i.innerHTML=o.join(""),0===r.length&&(i.style.display="none"),this.element.appendChild(i),this.element.appendChild(n),i.addEventListener((0,d.Le)(),(function(o){var a=(0,b.S)(o.target,"BUTTON");if(a){var l=a.getAttribute("data-type"),s=r.find((function(e){return(null==e?void 0:e.key)===l}));s?s.click(l):"mp-wechat"!==l&&"zhihu"!==l?(n.style.width="desktop"===l?"auto":"tablet"===l?"780px":"360px",n.scrollWidth>n.parentElement.clientWidth&&(n.style.width="auto"),t.render(e),i.querySelectorAll("button").forEach((function(e){e.classList.remove("vditor-preview__action--current")})),a.classList.add("vditor-preview__action--current")):t.copyToX(e,t.element.lastElementChild.cloneNode(!0),l)}}))}return e.prototype.render=function(e,t){var n=this;if(clearTimeout(this.mdTimeoutId),"none"!==this.element.style.display)if(t)this.element.lastElementChild.innerHTML=t;else if(""!==a(e).replace(/^[\s\uFEFF\xA0]+|[\s\uFEFF\xA0]+$/g,"")){var r=(new Date).getTime(),i=a(e);this.mdTimeoutId=window.setTimeout((function(){if(e.options.preview.url){var t=new XMLHttpRequest;t.open("POST",e.options.preview.url),t.setRequestHeader("Content-Type","application/json;charset=UTF-8"),t.onreadystatechange=function(){if(t.readyState===XMLHttpRequest.DONE)if(200===t.status){var o=JSON.parse(t.responseText);if(0!==o.code)return void e.tip.show(o.msg);e.options.preview.transform&&(o.data=e.options.preview.transform(o.data)),n.element.lastElementChild.innerHTML=o.data,n.afterRender(e,r)}else{var a=e.lute.Md2HTML(i);e.options.preview.transform&&(a=e.options.preview.transform(a)),n.element.lastElementChild.innerHTML=a,n.afterRender(e,r)}},t.send(JSON.stringify({markdownText:i}))}else{var o=e.lute.Md2HTML(i);e.options.preview.transform&&(o=e.options.preview.transform(o)),n.element.lastElementChild.innerHTML=o,n.afterRender(e,r)}}),e.options.preview.delay)}else this.element.lastElementChild.innerHTML="";else"renderPerformance"===this.element.getAttribute("data-type")&&e.tip.hide()},e.prototype.afterRender=function(e,t){e.options.preview.parse&&e.options.preview.parse(this.element);var n=(new Date).getTime()-t;(new Date).getTime()-t>2600?(e.tip.show(window.VditorI18n.performanceTip.replace("${x}",n.toString())),e.preview.element.setAttribute("data-type","renderPerformance")):"renderPerformance"===e.preview.element.getAttribute("data-type")&&(e.tip.hide(),e.preview.element.removeAttribute("data-type"));var r=e.preview.element.querySelector(".vditor-comment--focus");r&&r.classList.remove("vditor-comment--focus"),(0,S.O)(e.preview.element.lastElementChild),(0,T.s)(e.options.preview.hljs,e.preview.element.lastElementChild,e.options.cdn),(0,A.i)(e.preview.element.lastElementChild,e.options.cdn,e.options.theme),(0,C.P)(e.preview.element.lastElementChild,e.options.cdn),(0,L.v)(e.preview.element.lastElementChild,e.options.cdn),(0,k.p)(e.preview.element.lastElementChild,e.options.cdn,e.options.theme),(0,_.P)(e.preview.element.lastElementChild,e.options.cdn,e.options.theme),(0,x.B)(e.preview.element.lastElementChild,e.options.cdn),(0,E.Q)(e.preview.element.lastElementChild,e.options.cdn),(0,Rt.Y)(e.preview.element.lastElementChild);var i=e.preview.element,o=e.outline.render(e);""===o&&(o="[ToC]"),i.querySelectorAll('[data-type="toc-block"]').forEach((function(t){t.innerHTML=o,(0,M.H)(t,{cdn:e.options.cdn,math:e.options.preview.math})})),(0,M.H)(e.preview.element.lastElementChild,{cdn:e.options.cdn,math:e.options.preview.math})},e.prototype.copyToX=function(e,t,n){void 0===n&&(n="mp-wechat"),"zhihu"!==n?t.querySelectorAll(".katex-html .base").forEach((function(e){e.style.display="initial"})):t.querySelectorAll(".language-math").forEach((function(e){e.outerHTML='<img class="Formula-image" data-eeimg="true" src="//www.zhihu.com/equation?tex=" alt="'+e.getAttribute("data-math")+'\\" style="display: block; margin: 0 auto; max-width: 100%;">'})),t.style.backgroundColor="#fff",t.querySelectorAll("code").forEach((function(e){e.style.backgroundImage="none"})),this.element.append(t);var r=t.ownerDocument.createRange();r.selectNode(t),(0,N.Hc)(r),document.execCommand("copy"),this.element.lastElementChild.remove(),e.tip.show("已复制，可到"+("zhihu"===n?"知乎":"微信公众号平台")+"进行粘贴")},e}(),Bt=function(){function e(e){this.element=document.createElement("div"),this.element.className="vditor-resize vditor-resize--"+e.options.resize.position,this.element.innerHTML='<div><svg><use xlink:href="#vditor-icon-resize"></use></svg></div>',this.bindEvent(e)}return e.prototype.bindEvent=function(e){var t=this;this.element.addEventListener("mousedown",(function(n){var r=document,i=n.clientY,o=e.element.offsetHeight,a=63+e.element.querySelector(".vditor-toolbar").clientHeight;r.ondragstart=function(){return!1},window.captureEvents&&window.captureEvents(),t.element.classList.add("vditor-resize--selected"),r.onmousemove=function(t){"top"===e.options.resize.position?e.element.style.height=Math.max(a,o+(i-t.clientY))+"px":e.element.style.height=Math.max(a,o+(t.clientY-i))+"px",e.options.typewriterMode&&(e.sv.element.style.paddingBottom=e.sv.element.parentElement.offsetHeight/2+"px")},r.onmouseup=function(){e.options.resize.after&&e.options.resize.after(e.element.offsetHeight-o),window.captureEvents&&window.captureEvents(),r.onmousemove=null,r.onmouseup=null,r.ondragstart=null,r.onselectstart=null,r.onselect=null,t.element.classList.remove("vditor-resize--selected")}}))},e}(),qt=function(){function e(e){this.composingLock=!1,this.element=document.createElement("pre"),this.element.className="vditor-sv vditor-reset",this.element.setAttribute("placeholder",e.options.placeholder),this.element.setAttribute("contenteditable","true"),this.element.setAttribute("spellcheck","false"),this.bindEvent(e),we(e,this.element),ke(e,this.element),Me(e,this.element),Ae(e,this.element),Se(e,this.element),Ce(e,this.element,this.copy),Le(e,this.element,this.copy)}return e.prototype.copy=function(e,t){e.stopPropagation(),e.preventDefault(),e.clipboardData.setData("text/plain",be(t[t.currentMode].element))},e.prototype.bindEvent=function(e){var t=this;this.element.addEventListener("paste",(function(t){St(e,t,{pasteCode:function(e){document.execCommand("insertHTML",!1,e)}})})),this.element.addEventListener("scroll",(function(){if("block"===e.preview.element.style.display){var n=t.element.scrollTop,r=t.element.clientHeight,i=t.element.scrollHeight-parseFloat(t.element.style.paddingBottom||"0"),o=e.preview.element;o.scrollTop=n/r>.5?(n+r)*o.scrollHeight/i-r:n*o.scrollHeight/i}})),this.element.addEventListener("compositionstart",(function(e){t.composingLock=!0})),this.element.addEventListener("compositionend",(function(n){(0,d.vU)()||q(e,n),t.composingLock=!1})),this.element.addEventListener("input",(function(n){"deleteByDrag"!==n.inputType&&"insertFromDrop"!==n.inputType&&(t.composingLock||"‘"===n.data||"“"===n.data||"《"===n.data||(t.preventInput?t.preventInput=!1:q(e,n)))})),this.element.addEventListener("keyup",(function(t){t.isComposing||(0,d.yl)(t)||("Backspace"!==t.key&&"Delete"!==t.key||""===e.sv.element.innerHTML||1!==e.sv.element.childNodes.length||!e.sv.element.firstElementChild||"DIV"!==e.sv.element.firstElementChild.tagName||2!==e.sv.element.firstElementChild.childElementCount||""!==e.sv.element.firstElementChild.textContent&&"\n"!==e.sv.element.textContent?"Enter"===t.key&&Te(e):e.sv.element.innerHTML="")}))},e}(),Vt=function(){function e(){this.element=document.createElement("div"),this.element.className="vditor-tip"}return e.prototype.show=function(e,t){var n=this;if(void 0===t&&(t=6e3),this.element.className="vditor-tip vditor-tip--show",0===t)return this.element.innerHTML='<div class="vditor-tip__content">'+e+'\n<div class="vditor-tip__close">X</div></div>',void this.element.querySelector(".vditor-tip__close").addEventListener("click",(function(){n.hide()}));this.element.innerHTML='<div class="vditor-tip__content">'+e+"</div>",setTimeout((function(){n.hide()}),t)},e.prototype.hide=function(){this.element.className="vditor-messageElementtip",this.element.innerHTML=""},e}(),Ut=function(e,t){if(t.options.preview.mode!==e){switch(t.options.preview.mode=e,e){case"both":t.sv.element.style.display="block",t.preview.element.style.display="block",t.preview.render(t),u(t.toolbar.elements,["both"]);break;case"editor":t.sv.element.style.display="block",t.preview.element.style.display="none",c(t.toolbar.elements,["both"])}t.devtools&&t.devtools.renderEchart(t)}},Wt=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),zt=function(e){function t(t,n){var r=e.call(this,t,n)||this;return"both"===t.options.preview.mode&&r.element.children[0].classList.add("vditor-menu--current"),r.element.children[0].addEventListener((0,d.Le)(),(function(e){r.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)||(e.preventDefault(),"sv"===t.currentMode&&("both"===t.options.preview.mode?Ut("editor",t):Ut("both",t)))})),r}return Wt(t,e),t}(he),Gt=function(){this.element=document.createElement("div"),this.element.className="vditor-toolbar__br"},Kt=n(968),Ft=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),Zt=function(e){function t(t,n){var r=e.call(this,t,n)||this,o=r.element.children[0],a=document.createElement("div");a.className="vditor-hint"+(2===n.level?"":" vditor-panel--arrow");var l="";return i.g.CODE_THEME.forEach((function(e){l+="<button>"+e+"</button>"})),a.innerHTML='<div style="overflow: auto;max-height:'+window.innerHeight/2+'px">'+l+"</div>",a.addEventListener((0,d.Le)(),(function(e){"BUTTON"===e.target.tagName&&(v(t,["subToolbar"]),t.options.preview.hljs.style=e.target.textContent,(0,Kt.Y)(e.target.textContent,t.options.cdn),e.preventDefault(),e.stopPropagation())})),r.element.appendChild(a),g(t,a,o,n.level),r}return Ft(t,e),t}(he),Jt=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),Xt=function(e){function t(t,n){var r=e.call(this,t,n)||this,i=r.element.children[0],o=document.createElement("div");o.className="vditor-hint"+(2===n.level?"":" vditor-panel--arrow");var a="";return Object.keys(t.options.preview.theme.list).forEach((function(e){a+='<button data-type="'+e+'">'+t.options.preview.theme.list[e]+"</button>"})),o.innerHTML='<div style="overflow: auto;max-height:'+window.innerHeight/2+'px">'+a+"</div>",o.addEventListener((0,d.Le)(),(function(e){"BUTTON"===e.target.tagName&&(v(t,["subToolbar"]),t.options.preview.theme.current=e.target.getAttribute("data-type"),(0,V.Z)(t.options.preview.theme.current,t.options.preview.theme.path),e.preventDefault(),e.stopPropagation())})),r.element.appendChild(o),g(t,o,i,n.level),r}return Jt(t,e),t}(he),Yt=function(){function e(e){this.element=document.createElement("span"),this.element.className="vditor-counter vditor-tooltipped vditor-tooltipped__nw",this.render(e,"")}return e.prototype.render=function(e,t){var n=t.endsWith("\n")?t.length-1:t.length;if("text"===e.options.counter.type&&e[e.currentMode]){var r=e[e.currentMode].element.cloneNode(!0);r.querySelectorAll(".vditor-wysiwyg__preview").forEach((function(e){e.remove()})),n=r.textContent.length}"number"==typeof e.options.counter.max?(n>e.options.counter.max?this.element.className="vditor-counter vditor-counter--error":this.element.className="vditor-counter",this.element.innerHTML=n+"/"+e.options.counter.max):this.element.innerHTML=""+n,this.element.setAttribute("aria-label",e.options.counter.type),e.options.counter.after&&e.options.counter.after(n,{enable:e.options.counter.enable,max:e.options.counter.max,type:e.options.counter.type})},e}(),Qt=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),$t=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r.element.children[0].innerHTML=n.icon,r.element.children[0].addEventListener((0,d.Le)(),(function(e){e.preventDefault(),e.currentTarget.classList.contains(i.g.CLASS_MENU_DISABLED)||n.click(e,t)})),r}return Qt(t,e),t}(he),en=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),tn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r.element.firstElementChild.addEventListener((0,d.Le)(),(function(e){var n=r.element.firstElementChild;n.classList.contains(i.g.CLASS_MENU_DISABLED)||(e.preventDefault(),n.classList.contains("vditor-menu--current")?(n.classList.remove("vditor-menu--current"),t.devtools.element.style.display="none",W(t)):(n.classList.add("vditor-menu--current"),t.devtools.element.style.display="block",W(t),t.devtools.renderEchart(t)))})),r}return en(t,e),t}(he),nn=function(){this.element=document.createElement("div"),this.element.className="vditor-toolbar__divider"},rn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),on=function(e){function t(t,n){var r=e.call(this,t,n)||this,i=document.createElement("div");i.className="vditor-panel vditor-panel--arrow";var o="";Object.keys(t.options.hint.emoji).forEach((function(e){var n=t.options.hint.emoji[e];n.indexOf(".")>-1?o+='<button data-value=":'+e+': " data-key=":'+e+':"><img\ndata-value=":'+e+': " data-key=":'+e+':" class="vditor-emojis__icon" src="'+n+'"/></button>':o+='<button data-value="'+n+' "\n data-key="'+e+'"><span class="vditor-emojis__icon">'+n+"</span></button>"}));var a='<div class="vditor-emojis__tail">\n    <span class="vditor-emojis__tip"></span><span>'+(t.options.hint.emojiTail||"")+"</span>\n</div>";return i.innerHTML='<div class="vditor-emojis" style="max-height: '+("auto"===t.options.height?"auto":t.options.height-80)+'px">'+o+"</div>"+a,r.element.appendChild(i),g(t,i,r.element.children[0],n.level),r._bindEvent(t,i),r}return rn(t,e),t.prototype._bindEvent=function(e,t){t.querySelectorAll(".vditor-emojis button").forEach((function(n){n.addEventListener((0,d.Le)(),(function(r){r.preventDefault();var i=n.getAttribute("data-value"),o=(0,N.zh)(e),a=i;if("wysiwyg"===e.currentMode?a=e.lute.SpinVditorDOM(i):"ir"===e.currentMode&&(a=e.lute.SpinVditorIRDOM(i)),i.indexOf(":")>-1&&"sv"!==e.currentMode){var l=document.createElement("div");l.innerHTML=a,a=l.firstElementChild.firstElementChild.outerHTML+" ",(0,N.oC)(a,e)}else o.extractContents(),o.insertNode(document.createTextNode(i));o.collapse(!1),(0,N.Hc)(o),t.style.display="none",lt(e)})),n.addEventListener("mouseover",(function(e){"BUTTON"===e.target.tagName&&(t.querySelector(".vditor-emojis__tip").innerHTML=e.target.getAttribute("data-key"))}))}))},t}(he),an=function(e,t,n){var r=document.createElement("a");"download"in r?(r.download=n,r.style.display="none",r.href=URL.createObjectURL(new Blob([t])),document.body.appendChild(r),r.click(),r.remove()):e.tip.show(window.VditorI18n.downloadTip,0)},ln=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),sn=function(e){function t(t,n){var r=e.call(this,t,n)||this,i=r.element.children[0],o=document.createElement("div");return o.className="vditor-hint"+(2===n.level?"":" vditor-panel--arrow"),o.innerHTML='<button data-type="markdown">Markdown</button>\n<button data-type="pdf">PDF</button>\n<button data-type="html">HTML</button>',o.addEventListener((0,d.Le)(),(function(e){var n=e.target;if("BUTTON"===n.tagName){switch(n.getAttribute("data-type")){case"markdown":!function(e){var t=a(e);an(e,t,t.substr(0,10)+".md")}(t);break;case"pdf":!function(e){e.tip.show(window.VditorI18n.generate,3800);var t=document.querySelector("iframe");t.contentDocument.open(),t.contentDocument.write('<link rel="stylesheet" href="'+e.options.cdn+'/dist/index.css"/>\n<script src="'+e.options.cdn+'/dist/method.min.js"><\/script>\n<div id="preview"></div>\n<script>\nwindow.addEventListener("message", (e) => {\n  if(!e.data) {\n    return;\n  }\n  Vditor.preview(document.getElementById(\'preview\'), e.data, {\n    markdown: {\n      theme: "'+e.options.preview.theme+'"\n    },\n    hljs: {\n      style: "'+e.options.preview.hljs.style+'"\n    }\n  });\n  setTimeout(() => {\n        window.print();\n    }, 3600);\n}, false);\n<\/script>'),t.contentDocument.close(),setTimeout((function(){t.contentWindow.postMessage(a(e),"*")}),200)}(t);break;case"html":!function(e){var t=Dt(e),n='<html><head><link rel="stylesheet" type="text/css" href="'+e.options.cdn+'/dist/index.css"/>\n<script src="'+e.options.cdn+"/dist/js/i18n/"+e.options.lang+'.js"><\/script>\n<script src="'+e.options.cdn+'/dist/method.min.js"><\/script></head>\n<body><div class="vditor-reset" id="preview">'+t+"</div>\n<script>\n    const previewElement = document.getElementById('preview')\n    Vditor.setContentTheme('"+e.options.preview.theme.current+"', '"+e.options.preview.theme.path+"');\n    Vditor.codeRender(previewElement);\n    Vditor.highlightRender("+JSON.stringify(e.options.preview.hljs)+", previewElement, '"+e.options.cdn+"');\n    Vditor.mathRender(previewElement, {\n        cdn: '"+e.options.cdn+"',\n        math: "+JSON.stringify(e.options.preview.math)+",\n    });\n    Vditor.mermaidRender(previewElement, '"+e.options.cdn+"', '"+e.options.theme+"');\n    Vditor.flowchartRender(previewElement, '"+e.options.cdn+"');\n    Vditor.graphvizRender(previewElement, '"+e.options.cdn+"');\n    Vditor.chartRender(previewElement, '"+e.options.cdn+"', '"+e.options.theme+"');\n    Vditor.mindmapRender(previewElement, '"+e.options.cdn+"', '"+e.options.theme+"');\n    Vditor.abcRender(previewElement, '"+e.options.cdn+"');\n    Vditor.mediaRender(previewElement);\n    Vditor.speechRender(previewElement);\n<\/script>\n<script src=\""+e.options.cdn+"/dist/js/icons/"+e.options.icon+'.js"><\/script></body></html>';an(e,n,t.substr(0,10)+".html")}(t)}v(t,["subToolbar"]),e.preventDefault(),e.stopPropagation()}})),r.element.appendChild(o),g(t,o,i,n.level),r}return ln(t,e),t}(he),dn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),cn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r._bindEvent(t,n),r}return dn(t,e),t.prototype._bindEvent=function(e,t){this.element.children[0].addEventListener((0,d.Le)(),(function(n){n.preventDefault(),e.element.className.includes("vditor--fullscreen")?(t.level||(this.innerHTML=t.icon),e.element.style.zIndex="",document.body.style.overflow="",e.element.classList.remove("vditor--fullscreen"),Object.keys(e.toolbar.elements).forEach((function(t){var n=e.toolbar.elements[t].firstChild;n&&(n.className=n.className.replace("__s","__n"))})),e.counter&&(e.counter.element.className=e.counter.element.className.replace("__s","__n"))):(t.level||(this.innerHTML='<svg><use xlink:href="#vditor-icon-contract"></use></svg>'),e.element.style.zIndex=e.options.fullscreen.index.toString(),document.body.style.overflow="hidden",e.element.classList.add("vditor--fullscreen"),Object.keys(e.toolbar.elements).forEach((function(t){var n=e.toolbar.elements[t].firstChild;n&&(n.className=n.className.replace("__n","__s"))})),e.counter&&(e.counter.element.className=e.counter.element.className.replace("__n","__s"))),e.devtools&&e.devtools.renderEchart(e),t.click&&t.click(n,e),W(e),z(e)}))},t}(he),un=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),pn=function(e){function t(t,n){var r=e.call(this,t,n)||this,i=document.createElement("div");return i.className="vditor-hint vditor-panel--arrow",i.innerHTML='<button data-tag="h1" data-value="# ">'+window.VditorI18n.heading1+" "+(0,d.ns)("&lt;⌥⌘1>")+'</button>\n<button data-tag="h2" data-value="## ">'+window.VditorI18n.heading2+" &lt;"+(0,d.ns)("⌥⌘2")+'></button>\n<button data-tag="h3" data-value="### ">'+window.VditorI18n.heading3+" &lt;"+(0,d.ns)("⌥⌘3")+'></button>\n<button data-tag="h4" data-value="#### ">'+window.VditorI18n.heading4+" &lt;"+(0,d.ns)("⌥⌘4")+'></button>\n<button data-tag="h5" data-value="##### ">'+window.VditorI18n.heading5+" &lt;"+(0,d.ns)("⌥⌘5")+'></button>\n<button data-tag="h6" data-value="###### ">'+window.VditorI18n.heading6+" &lt;"+(0,d.ns)("⌥⌘6")+"></button>",r.element.appendChild(i),r._bindEvent(t,i),r}return un(t,e),t.prototype._bindEvent=function(e,t){var n=this.element.children[0];n.addEventListener((0,d.Le)(),(function(r){r.preventDefault(),n.classList.contains(i.g.CLASS_MENU_DISABLED)||(n.blur(),n.classList.contains("vditor-menu--current")?("wysiwyg"===e.currentMode?(te(e),X(e)):"ir"===e.currentMode&&Tt(e,""),n.classList.remove("vditor-menu--current")):(v(e,["subToolbar"]),t.style.display="block"))}));for(var r=0;r<6;r++)t.children.item(r).addEventListener((0,d.Le)(),(function(r){r.preventDefault(),"wysiwyg"===e.currentMode?(ee(e,r.target.getAttribute("data-tag")),X(e),n.classList.add("vditor-menu--current")):"ir"===e.currentMode?(Tt(e,r.target.getAttribute("data-value")),n.classList.add("vditor-menu--current")):Oe(e,r.target.getAttribute("data-value")),t.style.display="none"}))},t}(he),mn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),fn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r.element.children[0].addEventListener((0,d.Le)(),(function(e){e.preventDefault(),t.tip.show('<div style="margin-bottom:14px;font-size: 14px;line-height: 22px;min-width:300px;max-width: 360px;display: flex;">\n<div style="margin-top: 14px;flex: 1">\n    <div>Markdown 使用指南</div>\n    <ul style="list-style: none">\n        <li><a href="https://ld246.com/article/1583308420519" target="_blank">语法速查手册</a></li>\n        <li><a href="https://ld246.com/article/1583129520165" target="_blank">基础语法</a></li>\n        <li><a href="https://ld246.com/article/1583305480675" target="_blank">扩展语法</a></li>\n        <li><a href="https://ld246.com/article/1582778815353" target="_blank">键盘快捷键</a></li>\n    </ul>\n</div>\n<div style="margin-top: 14px;flex: 1">\n    <div>Vditor 支持</div>\n    <ul style="list-style: none">\n        <li><a href="https://github.com/Vanessa219/vditor/issues" target="_blank">Issues</a></li>\n        <li><a href="https://ld246.com/tag/vditor" target="_blank">官方讨论区</a></li>\n        <li><a href="https://ld246.com/article/1549638745630" target="_blank">开发手册</a></li>\n        <li><a href="https://ld246.com/guide/markdown" target="_blank">演示地址</a></li>\n    </ul>\n</div></div>',0)})),r}return mn(t,e),t}(he),hn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),vn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r.element.children[0].addEventListener((0,d.Le)(),(function(e){if(e.preventDefault(),!r.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)&&"sv"!==t.currentMode){var n=(0,N.zh)(t),o=(0,y.lG)(n.startContainer,"LI");o&&nt(t,o,n)}})),r}return hn(t,e),t}(he),gn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),yn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r.element.children[0].addEventListener((0,d.Le)(),(function(e){e.preventDefault(),t.tip.show('<div style="max-width: 520px; font-size: 14px;line-height: 22px;margin-bottom: 14px;">\n<p style="text-align: center;margin: 14px 0">\n    <em>下一代的 Markdown 编辑器，为未来而构建</em>\n</p>\n<div style="display: flex;margin-bottom: 14px;flex-wrap: wrap;align-items: center">\n    <img src="https://cdn.jsdelivr.net/npm/vditor/src/assets/images/logo.png" style="margin: 0 auto;height: 68px"/>\n    <div>&nbsp;&nbsp;</div>\n    <div style="flex: 1;min-width: 250px">\n        Vditor 是一款浏览器端的 Markdown 编辑器，支持所见即所得、即时渲染（类似 Typora）和分屏预览模式。\n        它使用 TypeScript 实现，支持原生 JavaScript 以及 Vue、React、Angular 和 Svelte 等框架。\n    </div>\n</div>\n<div style="display: flex;flex-wrap: wrap;">\n    <ul style="list-style: none;flex: 1;min-width:148px">\n        <li>\n        项目地址：<a href="https://b3log.org/vditor" target="_blank">b3log.org/vditor</a>\n        </li>\n        <li>\n        开源协议：MIT\n        </li>\n    </ul>\n    <ul style="list-style: none;margin-right: 18px">\n        <li>\n        组件版本：Vditor v'+i.H+" / Lute v"+Lute.Version+'\n        </li>\n        <li>\n        赞助捐赠：<a href="https://ld246.com/sponsor" target="_blank">https://ld246.com/sponsor</a>\n        </li>\n    </ul>\n</div>\n</div>',0)})),r}return gn(t,e),t}(he),bn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),wn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r.element.children[0].addEventListener((0,d.Le)(),(function(e){e.preventDefault(),r.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)||"sv"===t.currentMode||Je(t,"afterend")})),r}return bn(t,e),t}(he),En=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),kn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r.element.children[0].addEventListener((0,d.Le)(),(function(e){e.preventDefault(),r.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)||"sv"===t.currentMode||Je(t,"beforebegin")})),r}return En(t,e),t}(he),Sn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),Cn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r.element.children[0].addEventListener((0,d.Le)(),(function(e){if(e.preventDefault(),!r.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)&&"sv"!==t.currentMode){var n=(0,N.zh)(t),o=(0,y.lG)(n.startContainer,"LI");o&&rt(t,o,n,o.parentElement)}})),r}return Sn(t,e),t}(he),Ln=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),Tn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return t.options.outline&&r.element.firstElementChild.classList.add("vditor-menu--current"),r.element.children[0].addEventListener((0,d.Le)(),(function(e){e.preventDefault(),t.toolbar.elements.outline.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)||(t.options.outline.enable=!r.element.firstElementChild.classList.contains("vditor-menu--current"),t.outline.toggle(t,t.options.outline.enable))})),r}return Ln(t,e),t}(he),Mn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),An=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r._bindEvent(t),r}return Mn(t,e),t.prototype._bindEvent=function(e){var t=this;this.element.children[0].addEventListener((0,d.Le)(),(function(n){n.preventDefault();var r=t.element.firstElementChild;if(!r.classList.contains(i.g.CLASS_MENU_DISABLED)){var o=i.g.EDIT_TOOLBARS.concat(["both","edit-mode","devtools"]);r.classList.contains("vditor-menu--current")?(r.classList.remove("vditor-menu--current"),"sv"===e.currentMode?(e.sv.element.style.display="block","both"===e.options.preview.mode?e.preview.element.style.display="block":e.preview.element.style.display="none"):(e[e.currentMode].element.parentElement.style.display="block",e.preview.element.style.display="none"),p(e.toolbar.elements,o),e.outline.render(e)):(m(e.toolbar.elements,o),e.preview.element.style.display="block","sv"===e.currentMode?e.sv.element.style.display="none":e[e.currentMode].element.parentElement.style.display="none",e.preview.render(e),r.classList.add("vditor-menu--current"),v(e,["subToolbar","hint","popover"]),setTimeout((function(){e.outline.render(e)}),e.options.preview.delay+10)),W(e)}}))},t}(he),_n=function(){function e(e){var t;if(this.SAMPLE_RATE=5e3,this.isRecording=!1,this.readyFlag=!1,this.leftChannel=[],this.rightChannel=[],this.recordingLength=0,"undefined"!=typeof AudioContext)t=new AudioContext;else{if(!webkitAudioContext)return;t=new webkitAudioContext}this.DEFAULT_SAMPLE_RATE=t.sampleRate;var n=t.createGain();t.createMediaStreamSource(e).connect(n),this.recorder=t.createScriptProcessor(2048,2,1),this.recorder.onaudioprocess=null,n.connect(this.recorder),this.recorder.connect(t.destination),this.readyFlag=!0}return e.prototype.cloneChannelData=function(e,t){this.leftChannel.push(new Float32Array(e)),this.rightChannel.push(new Float32Array(t)),this.recordingLength+=2048},e.prototype.startRecordingNewWavFile=function(){this.readyFlag&&(this.isRecording=!0,this.leftChannel.length=this.rightChannel.length=0,this.recordingLength=0)},e.prototype.stopRecording=function(){this.isRecording=!1},e.prototype.buildWavFileBlob=function(){for(var e=this.mergeBuffers(this.leftChannel),t=this.mergeBuffers(this.rightChannel),n=new Float32Array(e.length),r=0;r<e.length;++r)n[r]=.5*(e[r]+t[r]);this.DEFAULT_SAMPLE_RATE>this.SAMPLE_RATE&&(n=this.downSampleBuffer(n,this.SAMPLE_RATE));var i=44+2*n.length,o=new ArrayBuffer(i),a=new DataView(o);this.writeUTFBytes(a,0,"RIFF"),a.setUint32(4,i,!0),this.writeUTFBytes(a,8,"WAVE"),this.writeUTFBytes(a,12,"fmt "),a.setUint32(16,16,!0),a.setUint16(20,1,!0),a.setUint16(22,1,!0),a.setUint32(24,this.SAMPLE_RATE,!0),a.setUint32(28,2*this.SAMPLE_RATE,!0),a.setUint16(32,2,!0),a.setUint16(34,16,!0);var l=2*n.length;this.writeUTFBytes(a,36,"data"),a.setUint32(40,l,!0);for(var s=n.length,d=44,c=0;c<s;c++)a.setInt16(d,32767*n[c],!0),d+=2;return new Blob([a],{type:"audio/wav"})},e.prototype.downSampleBuffer=function(e,t){if(t===this.DEFAULT_SAMPLE_RATE)return e;if(t>this.DEFAULT_SAMPLE_RATE)return e;for(var n=this.DEFAULT_SAMPLE_RATE/t,r=Math.round(e.length/n),i=new Float32Array(r),o=0,a=0;o<i.length;){for(var l=Math.round((o+1)*n),s=0,d=0,c=a;c<l&&c<e.length;c++)s+=e[c],d++;i[o]=s/d,o++,a=l}return i},e.prototype.mergeBuffers=function(e){for(var t=new Float32Array(this.recordingLength),n=0,r=e.length,i=0;i<r;++i){var o=e[i];t.set(o,n),n+=o.length}return t},e.prototype.writeUTFBytes=function(e,t,n){for(var r=n.length,i=0;i<r;i++)e.setUint8(t+i,n.charCodeAt(i))},e}(),xn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),Hn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return r._bindEvent(t),r}return xn(t,e),t.prototype._bindEvent=function(e){var t,n=this;this.element.children[0].addEventListener((0,d.Le)(),(function(r){if(r.preventDefault(),!n.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)){var o=e[e.currentMode].element;if(t)if(t.isRecording){t.stopRecording(),e.tip.hide();var a=new File([t.buildWavFileBlob()],"record"+(new Date).getTime()+".wav",{type:"video/webm"});Ve(e,[a]),n.element.children[0].classList.remove("vditor-menu--current")}else e.tip.show(window.VditorI18n.recording),o.setAttribute("contenteditable","false"),t.startRecordingNewWavFile(),n.element.children[0].classList.add("vditor-menu--current");else navigator.mediaDevices.getUserMedia({audio:!0}).then((function(r){(t=new _n(r)).recorder.onaudioprocess=function(e){if(t.isRecording){var n=e.inputBuffer.getChannelData(0),r=e.inputBuffer.getChannelData(1);t.cloneChannelData(n,r)}},t.startRecordingNewWavFile(),e.tip.show(window.VditorI18n.recording),o.setAttribute("contenteditable","false"),n.element.children[0].classList.add("vditor-menu--current")})).catch((function(){e.tip.show(window.VditorI18n["record-tip"])}))}}))},t}(he),Nn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),Dn=function(e){function t(t,n){var r=e.call(this,t,n)||this;return m({redo:r.element},["redo"]),r.element.children[0].addEventListener((0,d.Le)(),(function(e){e.preventDefault(),r.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)||t.undo.redo(t)})),r}return Nn(t,e),t}(he),On=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),In=function(e){function t(t,n){var r=e.call(this,t,n)||this;return m({undo:r.element},["undo"]),r.element.children[0].addEventListener((0,d.Le)(),(function(e){e.preventDefault(),r.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED)||t.undo.undo(t)})),r}return On(t,e),t}(he),jn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}(),Rn=function(e){function t(t,n){var r=e.call(this,t,n)||this,i='<input type="file"';return t.options.upload.multiple&&(i+=' multiple="multiple"'),t.options.upload.accept&&(i+=' accept="'+t.options.upload.accept+'"'),r.element.children[0].innerHTML=""+(n.icon||'<svg><use xlink:href="#vditor-icon-upload"></use></svg>')+i+">",r._bindEvent(t),r}return jn(t,e),t.prototype._bindEvent=function(e){var t=this;this.element.children[0].addEventListener((0,d.Le)(),(function(e){if(t.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED))return e.stopPropagation(),void e.preventDefault()})),this.element.querySelector("input").addEventListener("change",(function(n){if(t.element.firstElementChild.classList.contains(i.g.CLASS_MENU_DISABLED))return n.stopPropagation(),void n.preventDefault();0!==n.target.files.length&&Ve(e,n.target.files,n.target)}))},t}(he),Pn=function(){function e(e){var t=this,n=e.options;this.elements={},this.element=document.createElement("div"),this.element.className="vditor-toolbar",n.toolbar.forEach((function(n,r){var i=t.genItem(e,n,r);if(t.element.appendChild(i),n.toolbar){var o=document.createElement("div");o.className="vditor-hint vditor-panel--arrow",o.addEventListener((0,d.Le)(),(function(e){o.style.display="none"})),n.toolbar.forEach((function(n,i){n.level=2,o.appendChild(t.genItem(e,n,r+i))})),i.appendChild(o),g(e,o,i.children[0])}})),e.options.toolbarConfig.hide&&this.element.classList.add("vditor-toolbar--hide"),e.options.toolbarConfig.pin&&this.element.classList.add("vditor-toolbar--pin"),e.options.counter.enable&&(e.counter=new Yt(e),this.element.appendChild(e.counter.element))}return e.prototype.genItem=function(e,t,n){var r;switch(t.name){case"bold":case"italic":case"more":case"strike":case"line":case"quote":case"list":case"ordered-list":case"check":case"code":case"inline-code":case"link":case"table":r=new he(e,t);break;case"emoji":r=new on(e,t);break;case"headings":r=new pn(e,t);break;case"|":r=new nn;break;case"br":r=new Gt;break;case"undo":r=new In(e,t);break;case"redo":r=new Dn(e,t);break;case"help":r=new fn(e,t);break;case"both":r=new zt(e,t);break;case"preview":r=new An(e,t);break;case"fullscreen":r=new cn(e,t);break;case"upload":r=new Rn(e,t);break;case"record":r=new Hn(e,t);break;case"info":r=new yn(e,t);break;case"edit-mode":r=new ye(e,t);break;case"devtools":r=new tn(e,t);break;case"outdent":r=new Cn(e,t);break;case"indent":r=new vn(e,t);break;case"outline":r=new Tn(e,t);break;case"insert-after":r=new wn(e,t);break;case"insert-before":r=new kn(e,t);break;case"code-theme":r=new Zt(e,t);break;case"content-theme":r=new Xt(e,t);break;case"export":r=new sn(e,t);break;default:r=new $t(e,t)}if(r){var i=t.name;return"br"!==i&&"|"!==i||(i+=n),this.elements[i]=r.element,r.element}},e}(),Bn=n(694),qn=function(){function e(){this.stackSize=50,this.resetStack(),this.dmp=new Bn}return e.prototype.clearStack=function(e){this.resetStack(),this.resetIcon(e)},e.prototype.resetIcon=function(e){e.toolbar&&(this[e.currentMode].undoStack.length>1?p(e.toolbar.elements,["undo"]):m(e.toolbar.elements,["undo"]),0!==this[e.currentMode].redoStack.length?p(e.toolbar.elements,["redo"]):m(e.toolbar.elements,["redo"]))},e.prototype.undo=function(e){if("false"!==e[e.currentMode].element.getAttribute("contenteditable")&&!(this[e.currentMode].undoStack.length<2)){var t=this[e.currentMode].undoStack.pop();t&&(this[e.currentMode].redoStack.push(t),this.renderDiff(t,e),this[e.currentMode].hasUndo=!0,v(e,["hint"]))}},e.prototype.redo=function(e){if("false"!==e[e.currentMode].element.getAttribute("contenteditable")){var t=this[e.currentMode].redoStack.pop();t&&(this[e.currentMode].undoStack.push(t),this.renderDiff(t,e,!0))}},e.prototype.recordFirstPosition=function(e,t){if(0!==getSelection().rangeCount&&!(1!==this[e.currentMode].undoStack.length||0===this[e.currentMode].undoStack[0].length||this[e.currentMode].redoStack.length>0||(0,d.vU)()&&"Backspace"===t.key||(0,d.G6)())){var n=this.addCaret(e);n.replace("<wbr>","").replace(" vditor-ir__node--expand","")===this[e.currentMode].undoStack[0][0].diffs[0][1].replace("<wbr>","")&&(this[e.currentMode].undoStack[0][0].diffs[0][1]=n,this[e.currentMode].lastText=n)}},e.prototype.addToUndoStack=function(e){var t=this.addCaret(e,!0),n=this.dmp.diff_main(t,this[e.currentMode].lastText,!0),r=this.dmp.patch_make(t,this[e.currentMode].lastText,n);0===r.length&&this[e.currentMode].undoStack.length>0||(this[e.currentMode].lastText=t,this[e.currentMode].undoStack.push(r),this[e.currentMode].undoStack.length>this.stackSize&&this[e.currentMode].undoStack.shift(),this[e.currentMode].hasUndo&&(this[e.currentMode].redoStack=[],this[e.currentMode].hasUndo=!1,m(e.toolbar.elements,["redo"])),this[e.currentMode].undoStack.length>1&&p(e.toolbar.elements,["undo"]))},e.prototype.renderDiff=function(e,t,n){var r;if(void 0===n&&(n=!1),n){var i=this.dmp.patch_deepCopy(e).reverse();i.forEach((function(e){e.diffs.forEach((function(e){e[0]=-e[0]}))})),r=this.dmp.patch_apply(i,this[t.currentMode].lastText)[0]}else r=this.dmp.patch_apply(e,this[t.currentMode].lastText)[0];if(this[t.currentMode].lastText=r,t[t.currentMode].element.innerHTML=r,"sv"!==t.currentMode&&t[t.currentMode].element.querySelectorAll(".vditor-"+t.currentMode+"__preview[data-render='2']").forEach((function(e){H(e,t)})),t[t.currentMode].element.querySelector("wbr"))(0,N.ib)(t[t.currentMode].element,t[t.currentMode].element.ownerDocument.createRange()),Te(t);else{var o=getSelection().getRangeAt(0);o.setEndBefore(t[t.currentMode].element),o.collapse(!1)}lt(t,{enableAddUndoStack:!1,enableHint:!1,enableInput:!0}),pe(t),t[t.currentMode].element.querySelectorAll(".vditor-"+t.currentMode+"__preview[data-render='2']").forEach((function(e){H(e,t)})),this[t.currentMode].undoStack.length>1?p(t.toolbar.elements,["undo"]):m(t.toolbar.elements,["undo"]),0!==this[t.currentMode].redoStack.length?p(t.toolbar.elements,["redo"]):m(t.toolbar.elements,["redo"])},e.prototype.resetStack=function(){this.ir={hasUndo:!1,lastText:"",redoStack:[],undoStack:[]},this.sv={hasUndo:!1,lastText:"",redoStack:[],undoStack:[]},this.wysiwyg={hasUndo:!1,lastText:"",redoStack:[],undoStack:[]}},e.prototype.addCaret=function(e,t){var n;if(void 0===t&&(t=!1),0!==getSelection().rangeCount&&!e[e.currentMode].element.querySelector("wbr")){var r=getSelection().getRangeAt(0);if(e[e.currentMode].element.contains(r.startContainer)){n=r.cloneRange();var i=document.createElement("span");i.className="vditor-wbr",r.insertNode(i)}}e.ir.element.cloneNode(!0).querySelectorAll(".vditor-"+e.currentMode+"__preview[data-render='1']").forEach((function(e){(e.firstElementChild.classList.contains("language-echarts")||e.firstElementChild.classList.contains("language-plantuml")||e.firstElementChild.classList.contains("language-mindmap"))&&(e.firstElementChild.removeAttribute("_echarts_instance_"),e.firstElementChild.removeAttribute("data-processed"),e.firstElementChild.innerHTML=e.previousElementSibling.firstElementChild.innerHTML,e.setAttribute("data-render","2")),e.firstElementChild.classList.contains("language-math")&&(e.setAttribute("data-render","2"),e.firstElementChild.textContent=e.firstElementChild.getAttribute("data-math"),e.firstElementChild.removeAttribute("data-math"))}));var o=e[e.currentMode].element.innerHTML;return e[e.currentMode].element.querySelectorAll(".vditor-wbr").forEach((function(e){e.remove()})),t&&n&&(0,N.Hc)(n),o.replace('<span class="vditor-wbr"></span>',"<wbr>")},e}(),Vn=n(224),Un=function(){function e(e){this.defaultOptions={after:void 0,cache:{enable:!0},cdn:i.g.CDN,classes:{preview:""},comment:{enable:!1},counter:{enable:!1,type:"markdown"},debugger:!1,fullscreen:{index:90},height:"auto",hint:{delay:200,emoji:{"+1":"👍","-1":"👎",confused:"😕",eyes:"👀️",heart:"❤️",rocket:"🚀️",smile:"😄",tada:"🎉️"},emojiPath:i.g.CDN+"/dist/images/emoji",extend:[],parse:!0},icon:"ant",lang:"zh_CN",mode:"ir",outline:{enable:!1,position:"left"},placeholder:"",preview:{actions:["desktop","tablet","mobile","mp-wechat","zhihu"],delay:1e3,hljs:i.g.HLJS_OPTIONS,markdown:i.g.MARKDOWN_OPTIONS,math:i.g.MATH_OPTIONS,maxWidth:800,mode:"both",theme:i.g.THEME_OPTIONS},resize:{enable:!1,position:"bottom"},theme:"classic",toolbar:["emoji","headings","bold","italic","strike","link","|","list","ordered-list","check","outdent","indent","|","quote","line","code","inline-code","insert-before","insert-after","|","upload","record","table","|","undo","redo","|","fullscreen","edit-mode",{name:"more",toolbar:["both","code-theme","content-theme","export","outline","preview","devtools","info","help"]}],toolbarConfig:{hide:!1,pin:!1},typewriterMode:!1,undoDelay:800,upload:{extraData:{},fieldName:"file[]",filename:function(e){return e.replace(/\W/g,"")},linkToImgUrl:"",max:10485760,multiple:!0,url:"",withCredentials:!1},value:"",width:"auto"},this.options=e}return e.prototype.merge=function(){var e,t,n;this.options&&(this.options.toolbar?this.options.toolbar=this.mergeToolbar(this.options.toolbar):this.options.toolbar=this.mergeToolbar(this.defaultOptions.toolbar),(null===(t=null===(e=this.options.preview)||void 0===e?void 0:e.theme)||void 0===t?void 0:t.list)&&(this.defaultOptions.preview.theme.list=this.options.preview.theme.list),(null===(n=this.options.hint)||void 0===n?void 0:n.emoji)&&(this.defaultOptions.hint.emoji=this.options.hint.emoji),this.options.comment&&(this.defaultOptions.comment=this.options.comment));var r=(0,Vn.T)(this.defaultOptions,this.options);if(r.cache.enable&&!r.cache.id)throw new Error("need options.cache.id, see https://ld246.com/article/1549638745630#options");return r},e.prototype.mergeToolbar=function(e){var t=this,n=[{icon:'<svg><use xlink:href="#vditor-icon-export"></use></svg>',name:"export",tipPosition:"ne"},{hotkey:"⌘E",icon:'<svg><use xlink:href="#vditor-icon-emoji"></use></svg>',name:"emoji",tipPosition:"ne"},{hotkey:"⌘H",icon:'<svg><use xlink:href="#vditor-icon-headings"></use></svg>',name:"headings",tipPosition:"ne"},{hotkey:"⌘B",icon:'<svg><use xlink:href="#vditor-icon-bold"></use></svg>',name:"bold",prefix:"**",suffix:"**",tipPosition:"ne"},{hotkey:"⌘I",icon:'<svg><use xlink:href="#vditor-icon-italic"></use></svg>',name:"italic",prefix:"*",suffix:"*",tipPosition:"ne"},{hotkey:"⌘D",icon:'<svg><use xlink:href="#vditor-icon-strike"></use></svg>',name:"strike",prefix:"~~",suffix:"~~",tipPosition:"ne"},{hotkey:"⌘K",icon:'<svg><use xlink:href="#vditor-icon-link"></use></svg>',name:"link",prefix:"[",suffix:"](https://)",tipPosition:"n"},{name:"|"},{hotkey:"⌘L",icon:'<svg><use xlink:href="#vditor-icon-list"></use></svg>',name:"list",prefix:"* ",tipPosition:"n"},{hotkey:"⌘O",icon:'<svg><use xlink:href="#vditor-icon-ordered-list"></use></svg>',name:"ordered-list",prefix:"1. ",tipPosition:"n"},{hotkey:"⌘J",icon:'<svg><use xlink:href="#vditor-icon-check"></use></svg>',name:"check",prefix:"* [ ] ",tipPosition:"n"},{hotkey:"⇧⌘I",icon:'<svg><use xlink:href="#vditor-icon-outdent"></use></svg>',name:"outdent",tipPosition:"n"},{hotkey:"⇧⌘O",icon:'<svg><use xlink:href="#vditor-icon-indent"></use></svg>',name:"indent",tipPosition:"n"},{name:"|"},{hotkey:"⌘;",icon:'<svg><use xlink:href="#vditor-icon-quote"></use></svg>',name:"quote",prefix:"> ",tipPosition:"n"},{hotkey:"⇧⌘H",icon:'<svg><use xlink:href="#vditor-icon-line"></use></svg>',name:"line",prefix:"---",tipPosition:"n"},{hotkey:"⌘U",icon:'<svg><use xlink:href="#vditor-icon-code"></use></svg>',name:"code",prefix:"```",suffix:"\n```",tipPosition:"n"},{hotkey:"⌘G",icon:'<svg><use xlink:href="#vditor-icon-inline-code"></use></svg>',name:"inline-code",prefix:"`",suffix:"`",tipPosition:"n"},{hotkey:"⇧⌘B",icon:'<svg><use xlink:href="#vditor-icon-before"></use></svg>',name:"insert-before",tipPosition:"n"},{hotkey:"⇧⌘E",icon:'<svg><use xlink:href="#vditor-icon-after"></use></svg>',name:"insert-after",tipPosition:"n"},{name:"|"},{icon:'<svg><use xlink:href="#vditor-icon-upload"></use></svg>',name:"upload",tipPosition:"n"},{icon:'<svg><use xlink:href="#vditor-icon-record"></use></svg>',name:"record",tipPosition:"n"},{hotkey:"⌘M",icon:'<svg><use xlink:href="#vditor-icon-table"></use></svg>',name:"table",prefix:"| col1",suffix:" | col2 | col3 |\n| --- | --- | --- |\n|  |  |  |\n|  |  |  |",tipPosition:"n"},{name:"|"},{hotkey:"⌘Z",icon:'<svg><use xlink:href="#vditor-icon-undo"></use></svg>',name:"undo",tipPosition:"nw"},{hotkey:"⌘Y",icon:'<svg><use xlink:href="#vditor-icon-redo"></use></svg>',name:"redo",tipPosition:"nw"},{name:"|"},{icon:'<svg><use xlink:href="#vditor-icon-more"></use></svg>',name:"more",tipPosition:"e"},{hotkey:"⌘'",icon:'<svg><use xlink:href="#vditor-icon-fullscreen"></use></svg>',name:"fullscreen",tipPosition:"nw"},{icon:'<svg><use xlink:href="#vditor-icon-edit"></use></svg>',name:"edit-mode",tipPosition:"nw"},{hotkey:"⌘P",icon:'<svg><use xlink:href="#vditor-icon-both"></use></svg>',name:"both",tipPosition:"nw"},{icon:'<svg><use xlink:href="#vditor-icon-preview"></use></svg>',name:"preview",tipPosition:"nw"},{icon:'<svg><use xlink:href="#vditor-icon-align-center"></use></svg>',name:"outline",tipPosition:"nw"},{icon:'<svg><use xlink:href="#vditor-icon-theme"></use></svg>',name:"content-theme",tipPosition:"nw"},{icon:'<svg><use xlink:href="#vditor-icon-code-theme"></use></svg>',name:"code-theme",tipPosition:"nw"},{icon:'<svg><use xlink:href="#vditor-icon-bug"></use></svg>',name:"devtools",tipPosition:"nw"},{icon:'<svg><use xlink:href="#vditor-icon-info"></use></svg>',name:"info",tipPosition:"nw"},{icon:'<svg><use xlink:href="#vditor-icon-help"></use></svg>',name:"help",tipPosition:"nw"},{name:"br"}],r=[];return e.forEach((function(e){var i=e;n.forEach((function(t){"string"==typeof e&&t.name===e&&(i=t),"object"==typeof e&&t.name===e.name&&(i=Object.assign({},t,e))})),e.toolbar&&(i.toolbar=t.mergeToolbar(e.toolbar)),r.push(i)})),r},e}(),Wn=function(){function e(e){var t=this;this.composingLock=!1,this.commentIds=[];var n=document.createElement("div");n.className="vditor-wysiwyg",n.innerHTML='<pre class="vditor-reset" placeholder="'+e.options.placeholder+'"\n contenteditable="true" spellcheck="false"></pre>\n<div class="vditor-panel vditor-panel--none"></div>\n<div class="vditor-panel vditor-panel--none">\n    <button type="button" aria-label="'+window.VditorI18n.comment+'" class="vditor-icon vditor-tooltipped vditor-tooltipped__n">\n        <svg><use xlink:href="#vditor-icon-comment"></use></svg>\n    </button>\n</div>',this.element=n.firstElementChild,this.popover=n.firstElementChild.nextElementSibling,this.selectPopover=n.lastElementChild,this.bindEvent(e),we(e,this.element),Ee(e,this.element),ke(e,this.element),Me(e,this.element),Ae(e,this.element),Se(e,this.element),Ce(e,this.element,this.copy),Le(e,this.element,this.copy),e.options.comment.enable&&(this.selectPopover.querySelector("button").onclick=function(){var n,r,o=Lute.NewNodeID(),a=getSelection().getRangeAt(0),l=a.cloneRange(),s=a.extractContents(),d=!1,c=!1;s.childNodes.forEach((function(e,t){var i=!1;if(3===e.nodeType?i=!0:e.classList.contains("vditor-comment")?e.classList.contains("vditor-comment")&&e.setAttribute("data-cmtids",e.getAttribute("data-cmtids")+" "+o):i=!0,i)if(3!==e.nodeType&&"0"===e.getAttribute("data-block")&&0===t&&l.startOffset>0)e.innerHTML='<span class="vditor-comment" data-cmtids="'+o+'">'+e.innerHTML+"</span>",n=e;else if(3!==e.nodeType&&"0"===e.getAttribute("data-block")&&t===s.childNodes.length-1&&l.endOffset<l.endContainer.textContent.length)e.innerHTML='<span class="vditor-comment" data-cmtids="'+o+'">'+e.innerHTML+"</span>",r=e;else if(3!==e.nodeType&&"0"===e.getAttribute("data-block"))0===t?d=!0:t===s.childNodes.length-1&&(c=!0),e.innerHTML='<span class="vditor-comment" data-cmtids="'+o+'">'+e.innerHTML+"</span>";else{var a=document.createElement("span");a.classList.add("vditor-comment"),a.setAttribute("data-cmtids",o),e.parentNode.insertBefore(a,e),a.appendChild(e)}}));var u=(0,y.F9)(l.startContainer);u&&(n?(u.insertAdjacentHTML("beforeend",n.innerHTML),n.remove()):""===u.textContent.trim().replace(i.g.ZWSP,"")&&d&&u.remove());var p=(0,y.F9)(l.endContainer);p&&(r?(p.insertAdjacentHTML("afterbegin",r.innerHTML),r.remove()):""===p.textContent.trim().replace(i.g.ZWSP,"")&&c&&p.remove()),a.insertNode(s),e.options.comment.add(o,a.toString(),t.getComments(e,!0)),X(e,{enableAddUndoStack:!0,enableHint:!1,enableInput:!1}),t.hideComment()})}return e.prototype.getComments=function(e,t){var n=this;if(void 0===t&&(t=!1),"wysiwyg"!==e.currentMode||!e.options.comment.enable)return[];this.commentIds=[],this.element.querySelectorAll(".vditor-comment").forEach((function(e){n.commentIds=n.commentIds.concat(e.getAttribute("data-cmtids").split(" "))})),this.commentIds=Array.from(new Set(this.commentIds));var r=[];return t?(this.commentIds.forEach((function(e){r.push({id:e,top:n.element.querySelector('.vditor-comment[data-cmtids="'+e+'"]').offsetTop})})),r):void 0},e.prototype.triggerRemoveComment=function(e){var t,n,r;if("wysiwyg"===e.currentMode&&e.options.comment.enable&&e.wysiwyg.commentIds.length>0){var i=JSON.parse(JSON.stringify(this.commentIds));this.getComments(e);var o=(t=i,n=this.commentIds,r=new Set(n),t.filter((function(e){return!r.has(e)})));o.length>0&&e.options.comment.remove(o)}},e.prototype.showComment=function(){var e=(0,N.Ny)(this.element);this.selectPopover.setAttribute("style","left:"+e.left+"px;display:block;top:"+Math.max(-8,e.top-21)+"px")},e.prototype.hideComment=function(){this.selectPopover.setAttribute("style","display:none")},e.prototype.unbindListener=function(){window.removeEventListener("scroll",this.scrollListener)},e.prototype.copy=function(e,t){var n=getSelection().getRangeAt(0);if(""!==n.toString()){e.stopPropagation(),e.preventDefault();var r=(0,y.lG)(n.startContainer,"CODE"),i=(0,y.lG)(n.endContainer,"CODE");if(r&&i&&i.isSameNode(r)){var o="";return o="PRE"===r.parentElement.tagName?n.toString():"`"+n.toString()+"`",e.clipboardData.setData("text/plain",o),void e.clipboardData.setData("text/html","")}var a=(0,y.lG)(n.startContainer,"A"),l=(0,y.lG)(n.endContainer,"A");if(a&&l&&l.isSameNode(a)){var s=a.getAttribute("title")||"";return s&&(s=' "'+s+'"'),e.clipboardData.setData("text/plain","["+n.toString()+"]("+a.getAttribute("href")+s+")"),void e.clipboardData.setData("text/html","")}var d=document.createElement("div");d.appendChild(n.cloneContents()),e.clipboardData.setData("text/plain",t.lute.VditorDOM2Md(d.innerHTML).trim()),e.clipboardData.setData("text/html","")}},e.prototype.bindEvent=function(e){var t=this;this.unbindListener(),window.addEventListener("scroll",this.scrollListener=function(){if(v(e,["hint"]),"block"===t.popover.style.display&&"block"===t.selectPopover.style.display){var n=parseInt(t.popover.getAttribute("data-top"),10);if("auto"===e.options.height){if(e.options.toolbarConfig.pin){var r=Math.max(n,window.scrollY-e.element.offsetTop-8)+"px";"block"===t.popover.style.display&&(t.popover.style.top=r),"block"===t.selectPopover.style.display&&(t.selectPopover.style.top=r)}}else if(e.options.toolbarConfig.pin&&0===e.toolbar.element.getBoundingClientRect().top){var i=Math.max(window.scrollY-e.element.offsetTop-8,Math.min(n-e.wysiwyg.element.scrollTop,t.element.clientHeight-21))+"px";"block"===t.popover.style.display&&(t.popover.style.top=i),"block"===t.selectPopover.style.display&&(t.selectPopover.style.top=i)}}}),this.element.addEventListener("scroll",(function(){if(v(e,["hint"]),e.options.comment&&e.options.comment.enable&&e.options.comment.scroll&&e.options.comment.scroll(e.wysiwyg.element.scrollTop),"block"===t.popover.style.display){var n=parseInt(t.popover.getAttribute("data-top"),10)-e.wysiwyg.element.scrollTop,r=-8;e.options.toolbarConfig.pin&&0===e.toolbar.element.getBoundingClientRect().top&&(r=window.scrollY-e.element.offsetTop+r);var i=Math.max(r,Math.min(n,t.element.clientHeight-21))+"px";t.popover.style.top=i,t.selectPopover.style.top=i}})),this.element.addEventListener("paste",(function(t){St(e,t,{pasteCode:function(t){var n=(0,N.zh)(e),r=document.createElement("template");r.innerHTML=t,n.insertNode(r.content.cloneNode(!0));var i=(0,y.a1)(n.startContainer,"data-block","0");i?i.outerHTML=e.lute.SpinVditorDOM(i.outerHTML):e.wysiwyg.element.innerHTML=e.lute.SpinVditorDOM(e.wysiwyg.element.innerHTML),(0,N.ib)(e.wysiwyg.element,n)}})})),this.element.addEventListener("compositionstart",(function(){t.composingLock=!0})),this.element.addEventListener("compositionend",(function(n){var r=(0,b.W)(getSelection().getRangeAt(0).startContainer);r&&""===r.textContent?D(e):((0,d.vU)()||Ue(e,getSelection().getRangeAt(0).cloneRange(),n),t.composingLock=!1)})),this.element.addEventListener("input",(function(n){if("deleteByDrag"!==n.inputType&&"insertFromDrop"!==n.inputType)if(t.preventInput)t.preventInput=!1;else if(!t.composingLock&&"‘"!==n.data&&"“"!==n.data&&"《"!==n.data){var r=getSelection().getRangeAt(0),i=(0,y.F9)(r.startContainer);if(i||($(e,r),i=(0,y.F9)(r.startContainer)),i){for(var o=(0,N.im)(i,e.wysiwyg.element,r).start,a=!0,l=o-1;l>i.textContent.substr(0,o).lastIndexOf("\n");l--)if(" "!==i.textContent.charAt(l)&&"\t"!==i.textContent.charAt(l)){a=!1;break}0===o&&(a=!1);var s=!0;for(l=o-1;l<i.textContent.length;l++)if(" "!==i.textContent.charAt(l)&&"\n"!==i.textContent.charAt(l)){s=!1;break}var d=(0,b.W)(getSelection().getRangeAt(0).startContainer);d&&""===d.textContent&&(D(e),d.remove()),a&&"code-block"!==i.getAttribute("data-type")||s||at(i.innerHTML)||ot(i.innerHTML)&&i.previousElementSibling||Ue(e,r,n)}}})),this.element.addEventListener("click",(function(n){if("INPUT"===n.target.tagName){var r=n.target;return r.checked?r.setAttribute("checked","checked"):r.removeAttribute("checked"),t.preventInput=!0,void X(e)}if("IMG"!==n.target.tagName||n.target.parentElement.classList.contains("vditor-wysiwyg__preview")){"A"===n.target.tagName&&window.open(n.target.getAttribute("href"));var o=(0,N.zh)(e);if(n.target.isEqualNode(t.element)&&t.element.lastElementChild&&o.collapsed){var a=t.element.lastElementChild.getBoundingClientRect();n.y>a.top+a.height&&("P"===t.element.lastElementChild.tagName&&""===t.element.lastElementChild.textContent.trim().replace(i.g.ZWSP,"")?(o.selectNodeContents(t.element.lastElementChild),o.collapse(!1)):(t.element.insertAdjacentHTML("beforeend",'<p data-block="0">'+i.g.ZWSP+"<wbr></p>"),(0,N.ib)(t.element,o)))}ie(e);var l=(0,y.fb)(n.target,"vditor-wysiwyg__preview");l||(l=(0,y.fb)((0,N.zh)(e).startContainer,"vditor-wysiwyg__preview")),l&&ne(l,e),O(n,e)}else"link-ref"===n.target.getAttribute("data-type")?ae(e,n.target):function(e,t){var n=e.target;t.wysiwyg.popover.innerHTML="";var r=function(){n.setAttribute("src",o.value),n.setAttribute("alt",l.value),n.setAttribute("title",d.value)},i=document.createElement("span");i.setAttribute("aria-label",window.VditorI18n.imageURL),i.className="vditor-tooltipped vditor-tooltipped__n";var o=document.createElement("input");i.appendChild(o),o.className="vditor-input",o.setAttribute("placeholder",window.VditorI18n.imageURL),o.value=n.getAttribute("src")||"",o.oninput=function(){r()},o.onkeydown=function(e){re(t,e)};var a=document.createElement("span");a.setAttribute("aria-label",window.VditorI18n.alternateText),a.className="vditor-tooltipped vditor-tooltipped__n";var l=document.createElement("input");a.appendChild(l),l.className="vditor-input",l.setAttribute("placeholder",window.VditorI18n.alternateText),l.style.width="52px",l.value=n.getAttribute("alt")||"",l.oninput=function(){r()},l.onkeydown=function(e){re(t,e)};var s=document.createElement("span");s.setAttribute("aria-label",window.VditorI18n.title),s.className="vditor-tooltipped vditor-tooltipped__n";var d=document.createElement("input");s.appendChild(d),d.className="vditor-input",d.setAttribute("placeholder",window.VditorI18n.title),d.value=n.getAttribute("title")||"",d.oninput=function(){r()},d.onkeydown=function(e){re(t,e)},de(n,t),t.wysiwyg.popover.insertAdjacentElement("beforeend",i),t.wysiwyg.popover.insertAdjacentElement("beforeend",a),t.wysiwyg.popover.insertAdjacentElement("beforeend",s),oe(t,n)}(n,e)})),this.element.addEventListener("keyup",(function(t){if(!t.isComposing&&!(0,d.yl)(t)){"Enter"===t.key&&Te(e),"Backspace"!==t.key&&"Delete"!==t.key||""===e.wysiwyg.element.innerHTML||1!==e.wysiwyg.element.childNodes.length||!e.wysiwyg.element.firstElementChild||"P"!==e.wysiwyg.element.firstElementChild.tagName||0!==e.wysiwyg.element.firstElementChild.childElementCount||""!==e.wysiwyg.element.textContent&&"\n"!==e.wysiwyg.element.textContent||(e.wysiwyg.element.innerHTML="");var n=(0,N.zh)(e);if("Backspace"===t.key&&(0,d.vU)()&&"\n"===n.startContainer.textContent&&1===n.startOffset&&(n.startContainer.textContent=""),$(e,n),ie(e),"ArrowDown"===t.key||"ArrowRight"===t.key||"Backspace"===t.key||"ArrowLeft"===t.key||"ArrowUp"===t.key){"ArrowLeft"!==t.key&&"ArrowRight"!==t.key||e.hint.render(e);var r=(0,y.fb)(n.startContainer,"vditor-wysiwyg__preview");if(!r&&3!==n.startContainer.nodeType&&n.startOffset>0)(o=n.startContainer).classList.contains("vditor-wysiwyg__block")&&(r=o.lastElementChild);if(r)if("none"!==r.previousElementSibling.style.display){var i=r.previousElementSibling;if("PRE"===i.tagName&&(i=i.firstElementChild),"ArrowDown"===t.key||"ArrowRight"===t.key){var o,a=function(e){for(var t=e;t&&!t.nextSibling;)t=t.parentElement;return t.nextSibling}(o=r.parentElement);if(a&&3!==a.nodeType){var l=a.querySelector(".vditor-wysiwyg__preview");if(l)return void ne(l,e)}if(3===a.nodeType){for(;0===a.textContent.length&&a.nextSibling;)a=a.nextSibling;n.setStart(a,1)}else n.setStart(a.firstChild,0)}else n.selectNodeContents(i),n.collapse(!1)}else"ArrowDown"===t.key||"ArrowRight"===t.key?ne(r,e):ne(r,e,!1)}}}))},e}(),zn=function(){var e=function(t,n){return e=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)t.hasOwnProperty(n)&&(e[n]=t[n])},e(t,n)};return function(t,n){function r(){this.constructor=t}e(t,n),t.prototype=null===n?Object.create(n):(r.prototype=n.prototype,new r)}}();const Gn=function(e){function t(t,n){var r=e.call(this)||this;r.version=i.H,"string"==typeof t&&(n?n.cache?n.cache.id||(n.cache.id="vditor"+t):n.cache={id:"vditor"+t}:n={cache:{id:"vditor"+t}},t=document.getElementById(t));var o=new Un(n).merge();if(o.i18n)window.VditorI18n=o.i18n,r.init(t,o);else{if(!["en_US","ja_JP","ko_KR","ru_RU","zh_CN","zh_TW"].includes(o.lang))throw new Error("options.lang error, see https://ld246.com/article/1549638745630#options");var a="vditorI18nScript",s=a+o.lang;document.querySelectorAll('head script[id^="vditorI18nScript"]').forEach((function(e){e.id!==s&&document.head.removeChild(e)})),(0,l.G)(o.cdn+"/dist/js/i18n/"+o.lang+".js",s).then((function(){r.init(t,o)}))}return r}return zn(t,e),t.prototype.setTheme=function(e,t,n,r){this.vditor.options.theme=e,U(this.vditor),t&&(this.vditor.options.preview.theme.current=t,(0,V.Z)(t,r||this.vditor.options.preview.theme.path)),n&&(this.vditor.options.preview.hljs.style=n,(0,Kt.Y)(n,this.vditor.options.cdn))},t.prototype.getValue=function(){return a(this.vditor)},t.prototype.getCurrentMode=function(){return this.vditor.currentMode},t.prototype.focus=function(){"sv"===this.vditor.currentMode?this.vditor.sv.element.focus():"wysiwyg"===this.vditor.currentMode?this.vditor.wysiwyg.element.focus():"ir"===this.vditor.currentMode&&this.vditor.ir.element.focus()},t.prototype.blur=function(){"sv"===this.vditor.currentMode?this.vditor.sv.element.blur():"wysiwyg"===this.vditor.currentMode?this.vditor.wysiwyg.element.blur():"ir"===this.vditor.currentMode&&this.vditor.ir.element.blur()},t.prototype.disabled=function(){v(this.vditor,["subToolbar","hint","popover"]),m(this.vditor.toolbar.elements,i.g.EDIT_TOOLBARS.concat(["undo","redo","fullscreen","edit-mode"])),this.vditor[this.vditor.currentMode].element.setAttribute("contenteditable","false")},t.prototype.enable=function(){p(this.vditor.toolbar.elements,i.g.EDIT_TOOLBARS.concat(["undo","redo","fullscreen","edit-mode"])),this.vditor.undo.resetIcon(this.vditor),this.vditor[this.vditor.currentMode].element.setAttribute("contenteditable","true")},t.prototype.getSelection=function(){return"wysiwyg"===this.vditor.currentMode?be(this.vditor.wysiwyg.element):"sv"===this.vditor.currentMode?be(this.vditor.sv.element):"ir"===this.vditor.currentMode?be(this.vditor.ir.element):void 0},t.prototype.renderPreview=function(e){this.vditor.preview.render(this.vditor,e)},t.prototype.getCursorPosition=function(){return(0,N.Ny)(this.vditor[this.vditor.currentMode].element)},t.prototype.isUploading=function(){return this.vditor.upload.isUploading},t.prototype.clearCache=function(){localStorage.removeItem(this.vditor.options.cache.id)},t.prototype.disabledCache=function(){this.vditor.options.cache.enable=!1},t.prototype.enableCache=function(){if(!this.vditor.options.cache.id)throw new Error("need options.cache.id, see https://ld246.com/article/1549638745630#options");this.vditor.options.cache.enable=!0},t.prototype.html2md=function(e){return this.vditor.lute.HTML2Md(e)},t.prototype.exportJSON=function(e){return this.vditor.lute.RenderJSON(e)},t.prototype.getHTML=function(){return Dt(this.vditor)},t.prototype.tip=function(e,t){this.vditor.tip.show(e,t)},t.prototype.setPreviewMode=function(e){Ut(e,this.vditor)},t.prototype.deleteValue=function(){window.getSelection().isCollapsed||document.execCommand("delete",!1)},t.prototype.updateValue=function(e){document.execCommand("insertHTML",!1,e)},t.prototype.insertValue=function(e,t){void 0===t&&(t=!0);var n=(0,N.zh)(this.vditor);n.collapse(!0);var r=document.createElement("template");r.innerHTML=e,n.insertNode(r.content.cloneNode(!0)),"sv"===this.vditor.currentMode?(this.vditor.sv.preventInput=!0,t&&q(this.vditor)):"wysiwyg"===this.vditor.currentMode?(this.vditor.wysiwyg.preventInput=!0,t&&Ue(this.vditor,getSelection().getRangeAt(0))):"ir"===this.vditor.currentMode&&(this.vditor.ir.preventInput=!0,t&&j(this.vditor,getSelection().getRangeAt(0),!0))},t.prototype.setValue=function(e,t){var n=this;void 0===t&&(t=!1),"sv"===this.vditor.currentMode?(this.vditor.sv.element.innerHTML=this.vditor.lute.SpinVditorSVDOM(e),De(this.vditor,{enableAddUndoStack:!0,enableHint:!1,enableInput:!1})):"wysiwyg"===this.vditor.currentMode?me(this.vditor,e,{enableAddUndoStack:!0,enableHint:!1,enableInput:!1}):(this.vditor.ir.element.innerHTML=this.vditor.lute.Md2VditorIRDOM(e),this.vditor.ir.element.querySelectorAll(".vditor-ir__preview[data-render='2']").forEach((function(e){H(e,n.vditor)})),Lt(this.vditor,{enableAddUndoStack:!0,enableHint:!1,enableInput:!1})),this.vditor.outline.render(this.vditor),e||(v(this.vditor,["emoji","headings","submenu","hint"]),this.vditor.wysiwyg.popover&&(this.vditor.wysiwyg.popover.style.display="none"),this.clearCache()),t&&this.clearStack()},t.prototype.clearStack=function(){this.vditor.undo.clearStack(this.vditor),this.vditor.undo.addToUndoStack(this.vditor)},t.prototype.destroy=function(){this.vditor.element.innerHTML=this.vditor.originalInnerHTML,this.vditor.element.classList.remove("vditor"),this.vditor.element.removeAttribute("style"),document.getElementById("vditorIconScript").remove(),this.clearCache(),G(),this.vditor.wysiwyg.unbindListener()},t.prototype.getCommentIds=function(){return"wysiwyg"!==this.vditor.currentMode?[]:this.vditor.wysiwyg.getComments(this.vditor,!0)},t.prototype.hlCommentIds=function(e){if("wysiwyg"===this.vditor.currentMode){var t=function(t){t.classList.remove("vditor-comment--hover"),e.forEach((function(e){t.getAttribute("data-cmtids").indexOf(e)>-1&&t.classList.add("vditor-comment--hover")}))};this.vditor.wysiwyg.element.querySelectorAll(".vditor-comment").forEach((function(e){t(e)})),"none"!==this.vditor.preview.element.style.display&&this.vditor.preview.element.querySelectorAll(".vditor-comment").forEach((function(e){t(e)}))}},t.prototype.unHlCommentIds=function(e){if("wysiwyg"===this.vditor.currentMode){var t=function(t){e.forEach((function(e){t.getAttribute("data-cmtids").indexOf(e)>-1&&t.classList.remove("vditor-comment--hover")}))};this.vditor.wysiwyg.element.querySelectorAll(".vditor-comment").forEach((function(e){t(e)})),"none"!==this.vditor.preview.element.style.display&&this.vditor.preview.element.querySelectorAll(".vditor-comment").forEach((function(e){t(e)}))}},t.prototype.removeCommentIds=function(e){var t=this;if("wysiwyg"===this.vditor.currentMode){var n=function(e,n){var r=e.getAttribute("data-cmtids").split(" ");r.find((function(e,t){if(e===n)return r.splice(t,1),!0})),0===r.length?(e.outerHTML=e.innerHTML,(0,N.zh)(t.vditor).collapse(!0)):e.setAttribute("data-cmtids",r.join(" "))};e.forEach((function(e){t.vditor.wysiwyg.element.querySelectorAll(".vditor-comment").forEach((function(t){n(t,e)})),"none"!==t.vditor.preview.element.style.display&&t.vditor.preview.element.querySelectorAll(".vditor-comment").forEach((function(t){n(t,e)}))})),X(this.vditor,{enableAddUndoStack:!0,enableHint:!1,enableInput:!1})}},t.prototype.init=function(e,t){var n=this;this.vditor={currentMode:t.mode,element:e,hint:new Ht(t.hint.extend),lute:void 0,options:t,originalInnerHTML:e.innerHTML,outline:new jt(window.VditorI18n.outline),tip:new Vt},this.vditor.sv=new qt(this.vditor),this.vditor.undo=new qn,this.vditor.wysiwyg=new Wn(this.vditor),this.vditor.ir=new Nt(this.vditor),this.vditor.toolbar=new Pn(this.vditor),t.resize.enable&&(this.vditor.resize=new Bt(this.vditor)),this.vditor.toolbar.elements.devtools&&(this.vditor.devtools=new s),(t.upload.url||t.upload.handler)&&(this.vditor.upload=new qe),(0,l.G)(t._lutePath||t.cdn+"/dist/js/lute/lute.min.js","vditorLuteScript").then((function(){n.vditor.lute=(0,Ot.X)({autoSpace:n.vditor.options.preview.markdown.autoSpace,codeBlockPreview:n.vditor.options.preview.markdown.codeBlockPreview,emojiSite:n.vditor.options.hint.emojiPath,emojis:n.vditor.options.hint.emoji,fixTermTypo:n.vditor.options.preview.markdown.fixTermTypo,footnotes:n.vditor.options.preview.markdown.footnotes,headingAnchor:!1,inlineMathDigit:n.vditor.options.preview.math.inlineDigit,linkBase:n.vditor.options.preview.markdown.linkBase,linkPrefix:n.vditor.options.preview.markdown.linkPrefix,listStyle:n.vditor.options.preview.markdown.listStyle,mark:n.vditor.options.preview.markdown.mark,mathBlockPreview:n.vditor.options.preview.markdown.mathBlockPreview,paragraphBeginningSpace:n.vditor.options.preview.markdown.paragraphBeginningSpace,sanitize:n.vditor.options.preview.markdown.sanitize,toc:n.vditor.options.preview.markdown.toc}),n.vditor.preview=new Pt(n.vditor),function(e){e.element.innerHTML="",e.element.classList.add("vditor"),U(e),(0,V.Z)(e.options.preview.theme.current,e.options.preview.theme.path),"number"==typeof e.options.height&&(e.element.style.height=e.options.height+"px"),"number"==typeof e.options.minHeight&&(e.element.style.minHeight=e.options.minHeight+"px"),"number"==typeof e.options.width?e.element.style.width=e.options.width+"px":e.element.style.width=e.options.width,e.element.appendChild(e.toolbar.element);var t=document.createElement("div");if(t.className="vditor-content","left"===e.options.outline.position&&t.appendChild(e.outline.element),t.appendChild(e.wysiwyg.element.parentElement),t.appendChild(e.sv.element),t.appendChild(e.ir.element.parentElement),t.appendChild(e.preview.element),e.toolbar.elements.devtools&&t.appendChild(e.devtools.element),"right"===e.options.outline.position&&(e.outline.element.classList.add("vditor-outline--right"),t.appendChild(e.outline.element)),e.upload&&t.appendChild(e.upload.element),e.options.resize.enable&&t.appendChild(e.resize.element),t.appendChild(e.hint.element),t.appendChild(e.tip.element),e.element.appendChild(t),t.addEventListener("click",(function(){v(e,["subToolbar"])})),e.toolbar.elements.export&&e.element.insertAdjacentHTML("beforeend",'<iframe style="width: 100%;height: 0;border: 0"></iframe>'),ge(e,e.options.mode,Z(e)),document.execCommand("DefaultParagraphSeparator",!1,"p"),navigator.userAgent.indexOf("iPhone")>-1&&void 0!==window.visualViewport){var n=!1,r=function(t){n||(n=!0,requestAnimationFrame((function(){n=!1;var t=e.toolbar.element;t.style.transform="none",t.getBoundingClientRect().top<0&&(t.style.transform="translate(0, "+-t.getBoundingClientRect().top+"px)")})))};window.visualViewport.addEventListener("scroll",r),window.visualViewport.addEventListener("resize",r)}}(n.vditor),t.after&&t.after(),t.icon&&(0,l.J)(t.cdn+"/dist/js/icons/"+t.icon+".js","vditorIconScript")}))},t}(t.default)})(),r=r.default})()}));

/***/ })

/******/ 	});
/************************************************************************/
/******/ 	// The module cache
/******/ 	var __webpack_module_cache__ = {};
/******/ 	
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/ 		// Check if module is in cache
/******/ 		var cachedModule = __webpack_module_cache__[moduleId];
/******/ 		if (cachedModule !== undefined) {
/******/ 			return cachedModule.exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = __webpack_module_cache__[moduleId] = {
/******/ 			// no module.id needed
/******/ 			// no module.loaded needed
/******/ 			exports: {}
/******/ 		};
/******/ 	
/******/ 		// Execute the module function
/******/ 		__webpack_modules__[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/ 	
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/ 	
/************************************************************************/
/******/ 	/* webpack/runtime/compat get default export */
/******/ 	(() => {
/******/ 		// getDefaultExport function for compatibility with non-harmony modules
/******/ 		__webpack_require__.n = (module) => {
/******/ 			var getter = module && module.__esModule ?
/******/ 				() => (module['default']) :
/******/ 				() => (module);
/******/ 			__webpack_require__.d(getter, { a: getter });
/******/ 			return getter;
/******/ 		};
/******/ 	})();
/******/ 	
/******/ 	/* webpack/runtime/define property getters */
/******/ 	(() => {
/******/ 		// define getter functions for harmony exports
/******/ 		__webpack_require__.d = (exports, definition) => {
/******/ 			for(var key in definition) {
/******/ 				if(__webpack_require__.o(definition, key) && !__webpack_require__.o(exports, key)) {
/******/ 					Object.defineProperty(exports, key, { enumerable: true, get: definition[key] });
/******/ 				}
/******/ 			}
/******/ 		};
/******/ 	})();
/******/ 	
/******/ 	/* webpack/runtime/global */
/******/ 	(() => {
/******/ 		__webpack_require__.g = (function() {
/******/ 			if (typeof globalThis === 'object') return globalThis;
/******/ 			try {
/******/ 				return this || new Function('return this')();
/******/ 			} catch (e) {
/******/ 				if (typeof window === 'object') return window;
/******/ 			}
/******/ 		})();
/******/ 	})();
/******/ 	
/******/ 	/* webpack/runtime/hasOwnProperty shorthand */
/******/ 	(() => {
/******/ 		__webpack_require__.o = (obj, prop) => (Object.prototype.hasOwnProperty.call(obj, prop))
/******/ 	})();
/******/ 	
/******/ 	/* webpack/runtime/make namespace object */
/******/ 	(() => {
/******/ 		// define __esModule on exports
/******/ 		__webpack_require__.r = (exports) => {
/******/ 			if(typeof Symbol !== 'undefined' && Symbol.toStringTag) {
/******/ 				Object.defineProperty(exports, Symbol.toStringTag, { value: 'Module' });
/******/ 			}
/******/ 			Object.defineProperty(exports, '__esModule', { value: true });
/******/ 		};
/******/ 	})();
/******/ 	
/************************************************************************/
var __webpack_exports__ = {};
// This entry need to be wrapped in an IIFE because it need to be in strict mode.
(() => {
"use strict";
/*!*********************************************!*\
  !*** ./resources/js/plugins/Topic/topic.js ***!
  \*********************************************/
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var _babel_runtime_regenerator__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! @babel/runtime/regenerator */ "./node_modules/@babel/runtime/regenerator/index.js");
/* harmony import */ var _babel_runtime_regenerator__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(_babel_runtime_regenerator__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var vditor__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! vditor */ "./node_modules/vditor/dist/index.min.js");
/* harmony import */ var vditor__WEBPACK_IMPORTED_MODULE_1___default = /*#__PURE__*/__webpack_require__.n(vditor__WEBPACK_IMPORTED_MODULE_1__);
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! axios */ "./node_modules/axios/index.js");
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_2___default = /*#__PURE__*/__webpack_require__.n(axios__WEBPACK_IMPORTED_MODULE_2__);
/* harmony import */ var izitoast__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! izitoast */ "./node_modules/izitoast/dist/js/iziToast.js");
/* harmony import */ var izitoast__WEBPACK_IMPORTED_MODULE_3___default = /*#__PURE__*/__webpack_require__.n(izitoast__WEBPACK_IMPORTED_MODULE_3__);
/* harmony import */ var copy_to_clipboard__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! copy-to-clipboard */ "./node_modules/copy-to-clipboard/index.js");
/* harmony import */ var copy_to_clipboard__WEBPACK_IMPORTED_MODULE_4___default = /*#__PURE__*/__webpack_require__.n(copy_to_clipboard__WEBPACK_IMPORTED_MODULE_4__);
/* harmony import */ var sweetalert2__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! sweetalert2 */ "./node_modules/sweetalert2/dist/sweetalert2.all.js");
/* harmony import */ var sweetalert2__WEBPACK_IMPORTED_MODULE_5___default = /*#__PURE__*/__webpack_require__.n(sweetalert2__WEBPACK_IMPORTED_MODULE_5__);
/* harmony import */ var codemirror_src_edit_methods__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! codemirror/src/edit/methods */ "./node_modules/codemirror/src/edit/methods.js");


function asyncGeneratorStep(gen, resolve, reject, _next, _throw, key, arg) { try { var info = gen[key](arg); var value = info.value; } catch (error) { reject(error); return; } if (info.done) { resolve(value); } else { Promise.resolve(value).then(_next, _throw); } }

function _asyncToGenerator(fn) { return function () { var self = this, args = arguments; return new Promise(function (resolve, reject) { var gen = fn.apply(self, args); function _next(value) { asyncGeneratorStep(gen, resolve, reject, _next, _throw, "next", value); } function _throw(err) { asyncGeneratorStep(gen, resolve, reject, _next, _throw, "throw", err); } _next(undefined); }); }; }








if (document.getElementById("create-topic-vue")) {
  var create_topic_vue = {
    data: function data() {
      return {
        vditor: '',
        title: localStorage.getItem("topic_create_title"),
        edit: {
          mode: "ir",
          preview: {
            mode: "editor"
          }
        },
        options: {
          summary: ''
        },
        tag_selected: 1,
        tags: [{
          "text": "请选择",
          "value": "Default",
          "icons": "1"
        }],
        userAtList: [],
        topic_keywords: []
      };
    },
    methods: {
      edit_reply: function edit_reply() {
        var md = this.vditor.getSelection();
        this.vditor.updateValue("[reply]" + md + "[/reply]");
      },
      edit_mode: function edit_mode() {
        if (this.edit.mode === "ir") {
          this.edit.mode = "wysiwyg";
          this.init();
        } else {
          if (this.edit.mode === "wysiwyg") {
            this.edit.mode = "sv";
            this.edit.preview.mode = "editor";
            this.init();
          } else {
            if (this.edit.mode === "sv") {
              if (this.edit.preview.mode === "editor") {
                this.edit.mode = "sv";
                this.edit.preview.mode = "both";
                this.init();
              } else {
                if (this.edit.preview.mode === "both") {
                  this.edit.mode = "ir";
                  this.edit.preview.mode = "editor";
                  this.init();
                }
              }
            }
          }
        }

        izitoast__WEBPACK_IMPORTED_MODULE_3___default().show({
          title: 'success',
          message: '切换成功!',
          color: "#63ed7a",
          position: 'topRight',
          messageColor: '#ffffff',
          titleColor: '#ffffff'
        });
      },
      submit: function submit() {
        var _this = this;

        var html = this.vditor.getHTML();
        var markdown = this.vditor.getValue();
        var tags = this.tag_selected;
        var title = this.title;
        var summary = this.options.summary;

        if (!title) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '标题不能为空'
          });
          return;
        }

        if (!html || !markdown) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '正文内容不能为空'
          });
          return;
        }

        axios__WEBPACK_IMPORTED_MODULE_2___default().post("/topic/create", {
          _token: csrf_token,
          title: this.title,
          html: html,
          markdown: markdown,
          tag: tags,
          options_summary: summary
        }).then(function (r) {
          var data = r.data;

          if (!data.success) {
            data.result.forEach(function (value) {
              izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
                title: "error",
                message: value,
                position: "topRight",
                timeout: 10000
              });
            });
          } else {
            localStorage.removeItem("topic_create_title");
            localStorage.removeItem("topic_create_tag");

            _this.vditor.clearCache();

            data.result.forEach(function (value) {
              izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
                title: "success",
                message: value,
                position: "topRight",
                timeout: 10000
              });
            });
            setTimeout(function () {
              location.href = "/";
            }, 2000);
          }
        })["catch"](function (e) {
          console.error(e);
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '请求出错,详细查看控制台'
          });
        });
      },
      // 存为草稿
      draft: function draft() {
        var _this2 = this;

        var html = this.vditor.getHTML();
        var markdown = this.vditor.getValue();
        var tags = this.tag_selected;
        var title = this.title;
        var summary = this.options.summary;

        if (!title) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '标题不能为空'
          });
          return;
        }

        if (!html || !markdown) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '正文内容不能为空'
          });
          return;
        }

        axios__WEBPACK_IMPORTED_MODULE_2___default().post("/topic/create/draft", {
          _token: csrf_token,
          title: this.title,
          html: html,
          markdown: markdown,
          tag: tags,
          options_summary: summary
        }).then(function (r) {
          var data = r.data;

          if (!data.success) {
            data.result.forEach(function (value) {
              izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
                title: "error",
                message: value,
                position: "topRight",
                timeout: 10000
              });
            });
          } else {
            localStorage.removeItem("topic_create_title");
            localStorage.removeItem("topic_create_tag");

            _this2.vditor.clearCache();

            data.result.forEach(function (value) {
              izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
                title: "success",
                message: value,
                position: "topRight",
                timeout: 10000
              });
            });
            setTimeout(function () {
              location.href = "/";
            }, 2000);
          }
        })["catch"](function (e) {
          console.error(e);
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '请求出错,详细查看控制台'
          });
        });
      },
      // 帖子引用
      edit_with_topic: function edit_with_topic() {
        var _this3 = this;

        swal("输入帖子id或帖子链接:", {
          content: "input"
        }).then(function (value) {
          if (value) {
            var id;

            if (!/(^[1-9]\d*$)/.test(value)) {
              value = value.match(/\/(\S*)\.html/);

              if (value) {
                value = value[1];
              } else {
                return;
              }

              id = value.substring(value.lastIndexOf("/") + 1);
            } else {
              id = value;
            }

            var md = _this3.vditor.getSelection();

            copy_to_clipboard__WEBPACK_IMPORTED_MODULE_4___default()('[topic=' + id + ']' + md + '[/topic]');
            izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
              title: "Success",
              message: "短代码已复制,请在合适位置粘贴",
              position: "topRight"
            });
          }
        });
      },
      edit_with_files: function edit_with_files() {
        var _this4 = this;

        return _asyncToGenerator( /*#__PURE__*/_babel_runtime_regenerator__WEBPACK_IMPORTED_MODULE_0___default().mark(function _callee() {
          var _yield$Swal$fire, formValues, md;

          return _babel_runtime_regenerator__WEBPACK_IMPORTED_MODULE_0___default().wrap(function _callee$(_context) {
            while (1) {
              switch (_context.prev = _context.next) {
                case 0:
                  _context.next = 2;
                  return sweetalert2__WEBPACK_IMPORTED_MODULE_5___default().fire({
                    title: '添加附件',
                    html: '<label class="form-label">附件名 <b style="color:red">*</b></label> <input id="add-files-name" placeholder="附件名称" required class="swal-content__input">' + '<label class="form-label">下载链接 <b style="color:red">*</b></label> <input id="add-files-url" placeholder="下载链接" required class="swal-content__input">' + '<label class="form-label">下载密码</label><input id="add-files-pwd" placeholder="下载密码" class="swal-content__input">' + '<label class="form-label">解压密码</label><input id="add-files-unzip-pwd" placeholder="解压密码" class="swal-content__input">',
                    focusConfirm: false,
                    preConfirm: function preConfirm() {
                      return {
                        name: document.getElementById('add-files-name').value,
                        url: document.getElementById('add-files-url').value,
                        pwd: document.getElementById('add-files-pwd').value,
                        unzip: document.getElementById('add-files-unzip-pwd').value
                      };
                    }
                  });

                case 2:
                  _yield$Swal$fire = _context.sent;
                  formValues = _yield$Swal$fire.value;

                  if (!formValues) {
                    _context.next = 16;
                    break;
                  }

                  if (formValues.name) {
                    _context.next = 9;
                    break;
                  }

                  _context.next = 8;
                  return sweetalert2__WEBPACK_IMPORTED_MODULE_5___default().fire({
                    title: "附件名称不能为空!",
                    icon: "error"
                  });

                case 8:
                  return _context.abrupt("return");

                case 9:
                  if (formValues.url) {
                    _context.next = 13;
                    break;
                  }

                  _context.next = 12;
                  return sweetalert2__WEBPACK_IMPORTED_MODULE_5___default().fire({
                    title: "附件链接不能为空!",
                    icon: "error"
                  });

                case 12:
                  return _context.abrupt("return");

                case 13:
                  md = _this4.vditor.getSelection();
                  copy_to_clipboard__WEBPACK_IMPORTED_MODULE_4___default()('[file=' + formValues.name + ',' + formValues.url + ',' + formValues.pwd + ',' + formValues.unzip + ']' + md + '[/file]');
                  izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
                    title: "Success",
                    message: "短代码已复制,请在合适位置粘贴",
                    position: "topRight"
                  });

                case 16:
                case "end":
                  return _context.stop();
              }
            }
          }, _callee);
        }))();
      },
      edit_toc: function edit_toc() {
        var md = this.vditor.getValue();
        this.vditor.setValue("[toc]\n" + md);
      },
      init: function init() {
        var _this5 = this;

        // tags
        axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/topic/tags", {
          _token: csrf_token
        }).then(function (response) {
          _this5.tags = response.data;
        })["catch"](function (e) {
          console.error(e);
        }); // vditor

        this.vditor = new (vditor__WEBPACK_IMPORTED_MODULE_1___default())('content-vditor', {
          cdn: '/js/vditor',
          height: 400,
          toolbarConfig: {
            pin: true
          },
          cache: {
            enable: true,
            id: "create_topic"
          },
          preview: {
            markdown: {
              toc: true,
              mark: true,
              autoSpace: true
            }
          },
          mode: this.edit.mode,
          toolbar: ["emoji", "headings", "bold", "italic", "strike", "link", "|", "list", "ordered-list", "outdent", "indent", "|", "quote", "line", "code", "inline-code", "insert-before", "insert-after", "|", "upload", "record", "table", "|", "undo", "redo", "|", "fullscreen", "edit-mode"],
          counter: {
            "enable": true,
            "type": "已写字数"
          },
          hint: {
            extend: [{
              key: '@',
              hint: function hint(key) {
                return _this5.userAtList;
              }
            }, {
              key: '.',
              hint: function hint(key) {
                return _this5.topic_keywords;
              }
            }]
          },
          upload: {
            accept: 'image/*,.wav',
            token: csrf_token,
            url: imageUpUrl,
            linkToImgUrl: imageUpUrl,
            filename: function filename(name) {
              return name.replace(/[^(a-zA-Z0-9\u4e00-\u9fa5\.)]/g, '').replace(/[\?\\/:|<>\*\[\]\(\)\$%\{\}@~]/g, '').replace('/\\s/g', '');
            }
          },
          typewriterMode: true,
          placeholder: "请输入正文",
          after: function after() {
            axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/user/@user_list", {
              _token: csrf_token
            }).then(function (r) {
              _this5.userAtList = r.data;
            })["catch"](function (e) {
              swal({
                title: "获取本站用户列表失败,详细查看控制台",
                icon: "error"
              });
              console.error(e);
            });
            axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/topic/keywords", {
              _token: csrf_token
            }).then(function (r) {
              _this5.topic_keywords = r.data;
            })["catch"](function (e) {
              swal({
                title: "获取话题列表失败,详细查看控制台",
                icon: "error"
              });
              console.error(e);
            });
          },
          input: function input(md) {},
          select: function select(md) {}
        });
      }
    },
    mounted: function mounted() {
      if (localStorage.getItem("topic_create_tag")) {
        this.tag_selected = localStorage.getItem("topic_create_tag");
      }

      if (localStorage.getItem("topic_create_tag") || localStorage.getItem("topic_create_title")) {
        izitoast__WEBPACK_IMPORTED_MODULE_3___default().info({
          title: "Info",
          message: "已为您恢复上次编辑内容",
          position: 'topRight'
        });
      }

      this.init();
    },
    watch: {
      title: function title(_title) {
        localStorage.setItem("topic_create_title", _title);
      },
      tag_selected: function tag_selected(tag) {
        localStorage.setItem("topic_create_tag", tag);
      }
    }
  };
  Vue.createApp(create_topic_vue).mount("#create-topic-vue");
}

if (document.getElementById("topic-content")) {
  var previewElement = document.getElementById("topic-content");
  vditor__WEBPACK_IMPORTED_MODULE_1___default().mermaidRender(previewElement);
  vditor__WEBPACK_IMPORTED_MODULE_1___default().abcRender(previewElement);
  vditor__WEBPACK_IMPORTED_MODULE_1___default().chartRender(previewElement);
  vditor__WEBPACK_IMPORTED_MODULE_1___default().mindmapRender(previewElement);
  vditor__WEBPACK_IMPORTED_MODULE_1___default().graphvizRender(previewElement);
  vditor__WEBPACK_IMPORTED_MODULE_1___default().mathRender(previewElement);
  vditor__WEBPACK_IMPORTED_MODULE_1___default().mediaRender(previewElement);
  vditor__WEBPACK_IMPORTED_MODULE_1___default().highlightRender({
    lineNumber: true,
    enable: true
  }, previewElement);
  vditor__WEBPACK_IMPORTED_MODULE_1___default().flowchartRender(previewElement);
  vditor__WEBPACK_IMPORTED_MODULE_1___default().plantumlRender(previewElement);
} // 编辑帖子


if (document.getElementById("edit-topic-vue")) {
  var edit_topic_vue = {
    data: function data() {
      return {
        topic_id: topic_id,
        vditor: '',
        title: '',
        edit: {
          mode: "ir",
          preview: {
            mode: "editor"
          }
        },
        options: {
          summary: ''
        },
        tag_selected: 1,
        tags: [{
          "text": "请选择",
          "value": "Default",
          "icons": "1"
        }],
        userAtList: [],
        topic_keywords: []
      };
    },
    methods: {
      edit_reply: function edit_reply() {
        var md = this.vditor.getSelection();
        this.vditor.updateValue("[reply]" + md + "[/reply]");
      },
      edit_with_files: function edit_with_files() {
        var _this6 = this;

        return _asyncToGenerator( /*#__PURE__*/_babel_runtime_regenerator__WEBPACK_IMPORTED_MODULE_0___default().mark(function _callee2() {
          var _yield$Swal$fire2, formValues, md;

          return _babel_runtime_regenerator__WEBPACK_IMPORTED_MODULE_0___default().wrap(function _callee2$(_context2) {
            while (1) {
              switch (_context2.prev = _context2.next) {
                case 0:
                  _context2.next = 2;
                  return sweetalert2__WEBPACK_IMPORTED_MODULE_5___default().fire({
                    title: '添加附件',
                    html: '<label class="form-label">附件名 <b style="color:red">*</b></label> <input id="add-files-name" placeholder="附件名称" required class="swal-content__input">' + '<label class="form-label">下载链接 <b style="color:red">*</b></label> <input id="add-files-url" placeholder="下载链接" required class="swal-content__input">' + '<label class="form-label">下载密码</label><input id="add-files-pwd" placeholder="下载密码" class="swal-content__input">' + '<label class="form-label">解压密码</label><input id="add-files-unzip-pwd" placeholder="解压密码" class="swal-content__input">',
                    focusConfirm: false,
                    preConfirm: function preConfirm() {
                      return {
                        name: document.getElementById('add-files-name').value,
                        url: document.getElementById('add-files-url').value,
                        pwd: document.getElementById('add-files-pwd').value,
                        unzip: document.getElementById('add-files-unzip-pwd').value
                      };
                    }
                  });

                case 2:
                  _yield$Swal$fire2 = _context2.sent;
                  formValues = _yield$Swal$fire2.value;

                  if (!formValues) {
                    _context2.next = 16;
                    break;
                  }

                  if (formValues.name) {
                    _context2.next = 9;
                    break;
                  }

                  _context2.next = 8;
                  return sweetalert2__WEBPACK_IMPORTED_MODULE_5___default().fire({
                    title: "附件名称不能为空!",
                    icon: "error"
                  });

                case 8:
                  return _context2.abrupt("return");

                case 9:
                  if (formValues.url) {
                    _context2.next = 13;
                    break;
                  }

                  _context2.next = 12;
                  return sweetalert2__WEBPACK_IMPORTED_MODULE_5___default().fire({
                    title: "附件链接不能为空!",
                    icon: "error"
                  });

                case 12:
                  return _context2.abrupt("return");

                case 13:
                  md = _this6.vditor.getSelection();
                  copy_to_clipboard__WEBPACK_IMPORTED_MODULE_4___default()('[file=' + formValues.name + ',' + formValues.url + ',' + formValues.pwd + ',' + formValues.unzip + ']' + md + '[/file]');
                  izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
                    title: "Success",
                    message: "短代码已复制,请在合适位置粘贴",
                    position: "topRight"
                  });

                case 16:
                case "end":
                  return _context2.stop();
              }
            }
          }, _callee2);
        }))();
      },
      edit_mode: function edit_mode() {
        if (this.edit.mode === "ir") {
          this.edit.mode = "wysiwyg";
          this.init();
        } else {
          if (this.edit.mode === "wysiwyg") {
            this.edit.mode = "sv";
            this.edit.preview.mode = "editor";
            this.init();
          } else {
            if (this.edit.mode === "sv") {
              if (this.edit.preview.mode === "editor") {
                this.edit.mode = "sv";
                this.edit.preview.mode = "both";
                this.init();
              } else {
                if (this.edit.preview.mode === "both") {
                  this.edit.mode = "ir";
                  this.edit.preview.mode = "editor";
                  this.init();
                }
              }
            }
          }
        }

        izitoast__WEBPACK_IMPORTED_MODULE_3___default().show({
          title: 'success',
          message: '切换成功!',
          color: "#63ed7a",
          position: 'topRight',
          messageColor: '#ffffff',
          titleColor: '#ffffff'
        });
      },
      submit: function submit() {
        var _this7 = this;

        var html = this.vditor.getHTML();
        var markdown = this.vditor.getValue();
        var tags = this.tag_selected;
        var title = this.title;
        var summary = this.options.summary;

        if (!title) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '标题不能为空'
          });
          return;
        }

        if (!html || !markdown) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '正文内容不能为空'
          });
          return;
        }

        axios__WEBPACK_IMPORTED_MODULE_2___default().post("/topic/edit", {
          _token: csrf_token,
          topic_id: this.topic_id,
          title: this.title,
          html: html,
          markdown: markdown,
          tag: tags,
          summary: summary
        }).then(function (r) {
          var data = r.data;

          if (!data.success) {
            data.result.forEach(function (value) {
              izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
                title: "error",
                message: value,
                position: "topRight",
                timeout: 10000
              });
            });
          } else {
            _this7.vditor.clearCache();

            data.result.forEach(function (value) {
              izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
                title: "success",
                message: value,
                position: "topRight",
                timeout: 10000
              });
            });
            setTimeout(function () {
              location.href = "/" + this.topic_id + ".html";
            }, 2000);
          }
        })["catch"](function (e) {
          console.error(e);
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '请求出错,详细查看控制台'
          });
        });
      },
      // 存为草稿
      draft: function draft() {
        var _this8 = this;

        var html = this.vditor.getHTML();
        var markdown = this.vditor.getValue();
        var tags = this.tag_selected;
        var title = this.title;
        var summary = this.options.summary;

        if (!title) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '标题不能为空'
          });
          return;
        }

        if (!html || !markdown) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '正文内容不能为空'
          });
          return;
        }

        axios__WEBPACK_IMPORTED_MODULE_2___default().post("/topic/edit/draft", {
          _token: csrf_token,
          topic_id: this.topic_id,
          title: this.title,
          html: html,
          markdown: markdown,
          tag: tags,
          summary: summary
        }).then(function (r) {
          var data = r.data;

          if (!data.success) {
            data.result.forEach(function (value) {
              izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
                title: "error",
                message: value,
                position: "topRight",
                timeout: 10000
              });
            });
          } else {
            _this8.vditor.clearCache();

            data.result.forEach(function (value) {
              izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
                title: "success",
                message: value,
                position: "topRight",
                timeout: 10000
              });
            });
            setTimeout(function () {
              location.href = "/user/draft";
            }, 2000);
          }
        })["catch"](function (e) {
          console.error(e);
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '请求出错,详细查看控制台'
          });
        });
      },
      // 帖子引用
      edit_with_topic: function edit_with_topic() {
        var _this9 = this;

        swal("输入帖子id或帖子链接:", {
          content: "input"
        }).then(function (value) {
          if (value) {
            var id;

            if (!/(^[1-9]\d*$)/.test(value)) {
              value = value.match(/\/(\S*)\.html/);

              if (value) {
                value = value[1];
              } else {
                return;
              }

              id = value.substring(value.lastIndexOf("/") + 1);
            } else {
              id = value;
            }

            var md = _this9.vditor.getSelection();

            copy_to_clipboard__WEBPACK_IMPORTED_MODULE_4___default()('[topic=' + id + ']' + md + '[/topic]');
            izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
              title: "Success",
              message: "短代码已复制,请在合适位置粘贴",
              position: "topRight"
            });
          }
        });
      },
      edit_toc: function edit_toc() {
        var md = this.vditor.getValue();
        this.vditor.setValue("[toc]\n" + md);
      },
      init: function init() {
        var _this10 = this;

        // tags
        axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/topic/tags", {
          _token: csrf_token
        }).then(function (response) {
          _this10.tags = response.data;
        })["catch"](function (e) {
          console.error(e);
        }); // vditor

        this.vditor = new (vditor__WEBPACK_IMPORTED_MODULE_1___default())('content-vditor', {
          cdn: '/js/vditor',
          height: 400,
          toolbarConfig: {
            pin: true
          },
          cache: {
            enable: false
          },
          preview: {
            markdown: {
              toc: true,
              mark: true,
              autoSpace: true
            }
          },
          mode: this.edit.mode,
          toolbar: ["emoji", "headings", "bold", "italic", "strike", "link", "|", "list", "ordered-list", "outdent", "indent", "|", "quote", "line", "code", "inline-code", "insert-before", "insert-after", "|", "upload", "record", "table", "|", "undo", "redo", "|", "fullscreen", "edit-mode"],
          counter: {
            "enable": true,
            "type": "已写字数"
          },
          hint: {
            extend: [{
              key: '@',
              hint: function hint(key) {
                return _this10.userAtList;
              }
            }, {
              key: '.',
              hint: function hint(key) {
                return _this10.topic_keywords;
              }
            }]
          },
          upload: {
            accept: 'image/*,.wav',
            token: csrf_token,
            url: imageUpUrl,
            linkToImgUrl: imageUpUrl,
            filename: function filename(name) {
              return name.replace(/[^(a-zA-Z0-9\u4e00-\u9fa5\.)]/g, '').replace(/[\?\\/:|<>\*\[\]\(\)\$%\{\}@~]/g, '').replace('/\\s/g', '');
            }
          },
          typewriterMode: true,
          placeholder: "请输入正文",
          after: function after() {
            axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/user/@user_list", {
              _token: csrf_token
            }).then(function (r) {
              _this10.userAtList = r.data;
            })["catch"](function (e) {
              swal({
                title: "获取本站用户列表失败,详细查看控制台",
                icon: "error"
              });
              console.error(e);
            });
            axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/topic/keywords", {
              _token: csrf_token
            }).then(function (r) {
              _this10.topic_keywords = r.data;
            })["catch"](function (e) {
              swal({
                title: "获取话题列表失败,详细查看控制台",
                icon: "error"
              });
              console.error(e);
            });
            axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/topic/topic.data", {
              _token: csrf_token,
              topic_id: _this10.topic_id
            }).then(function (r) {
              var data = r.data;

              if (!data.success) {
                izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
                  title: 'Error',
                  position: 'topRight',
                  message: data.result.msg
                });
              } else {
                _this10.setTopicValue(data.result);
              }
            })["catch"](function (e) {
              console.error(e);
              izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
                title: 'Error',
                position: 'topRight',
                message: '请求出错,详细查看控制台'
              });
            });
          },
          input: function input(md) {},
          select: function select(md) {}
        });
      },
      setTopicValue: function setTopicValue(data) {
        this.title = data.title;
        this.vditor.setValue(data.markdown);
        this.tag_selected = data.tag.id;
        this.options.summary = data.options.summary;
      }
    },
    mounted: function mounted() {
      this.init();
    },
    watch: {
      title: function title(_title2) {},
      tag_selected: function tag_selected(tag) {}
    }
  };
  Vue.createApp(edit_topic_vue).mount("#edit-topic-vue");
} // 对帖子页面的操作


$(function () {
  // 精华
  $('a[core-click="topic-essence"]').click(function () {
    var topic_id = $(this).attr("topic-id");
    swal({
      title: "精华指数,数字越大排名越靠前",
      content: {
        element: "input",
        attributes: {
          type: "number",
          max: 999,
          min: 1
        }
      }
    }).then(function (r) {
      if (r && !isNaN(r) && r >= 0) {
        axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/topic/set.topic.essence", {
          _token: csrf_token,
          topic_id: topic_id,
          zhishu: r
        }).then(function (r) {
          var data = r.data;

          if (data.success) {
            izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
              title: 'Success',
              position: 'topRight',
              message: data.result.msg
            });
          } else {
            izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
              title: 'Error',
              position: 'topRight',
              message: data.result.msg
            });
          }
        })["catch"](function (e) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '请求出错,详细查看控制台'
          });
          console.error(e);
        });
      }
    });
  }); // 置顶

  $('a[core-click="topic-topping"]').click(function () {
    var topic_id = $(this).attr("topic-id");
    swal({
      title: "置顶指数,数字越大排名越靠前",
      content: {
        element: "input",
        attributes: {
          type: "number",
          max: 999,
          min: 1
        }
      }
    }).then(function (r) {
      if (r && !isNaN(r) && r >= 0) {
        axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/topic/set.topic.topping", {
          _token: csrf_token,
          topic_id: topic_id,
          zhishu: r
        }).then(function (r) {
          var data = r.data;

          if (data.success) {
            izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
              title: 'Success',
              position: 'topRight',
              message: data.result.msg
            });
          } else {
            izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
              title: 'Error',
              position: 'topRight',
              message: data.result.msg
            });
          }
        })["catch"](function (e) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '请求出错,详细查看控制台'
          });
          console.error(e);
        });
      }
    });
  }); // 删除

  $('a[core-click="topic-delete"]').click(function () {
    var topic_id = $(this).attr("topic-id");
    swal({
      title: "确定要删除此贴吗? 删除后不可恢复",
      buttons: ["取消", "确定"]
    }).then(function (r) {
      if (r === true) {
        axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/topic/set.topic.delete", {
          _token: csrf_token,
          topic_id: topic_id,
          zhishu: r
        }).then(function (r) {
          var data = r.data;

          if (data.success) {
            izitoast__WEBPACK_IMPORTED_MODULE_3___default().success({
              title: 'Success',
              position: 'topRight',
              message: data.result.msg
            });
          } else {
            izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
              title: 'Error',
              position: 'topRight',
              message: data.result.msg
            });
          }
        })["catch"](function (e) {
          izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
            title: 'Error',
            position: 'topRight',
            message: '请求出错,详细查看控制台'
          });
          console.error(e);
        });
      }
    });
  });
});
var author = {
  data: function data() {
    return {
      'user': {
        'city': null
      }
    };
  },
  mounted: function mounted() {
    this.getUserCity();
  },
  methods: {
    // 获取作者所在城市
    getUserCity: function getUserCity() {
      var _this11 = this;

      axios__WEBPACK_IMPORTED_MODULE_2___default().post("/api/topic/get.user", {
        _token: csrf_token,
        topic_id: topic_id
      }).then(function (r) {
        _this11.user = r.data.result;
      })["catch"](function (e) {
        izitoast__WEBPACK_IMPORTED_MODULE_3___default().error({
          title: 'Error',
          message: "请求出错,详细查看控制台",
          position: "topRight"
        });
        console.error(e);
      });
    }
  }
};
Vue.createApp(topic).mount('#author'); // 加载评论作者IP归属地

$(function () {
  var comments = [];
  $('small[comment-type="ip"]').each(function () {
    comments.push($(this).attr("comment-id"));
  });

  if (comments.length > 0) {
    axios__WEBPACK_IMPORTED_MODULE_2___default().post('/api/comment/get.user.ip', {
      _token: csrf_token,
      comments: comments
    }).then(function (r) {
      var data = r.data;
      data = data.result;
      data.forEach(function (v) {
        $('small[comment-id="' + v.comment_id + '"]').text(v.text);
      });
    });
  }
}); // 加载帖子更新记录作者IP归属地

$(function () {
  var updateds = [];
  $('span[topic-type="updated_ip"]').each(function () {
    updateds.push($(this).attr("updated-id"));
  });

  if (updateds.length > 0) {
    axios__WEBPACK_IMPORTED_MODULE_2___default().post('/api/topic/get.updated.user.ip', {
      _token: csrf_token,
      updateds: updateds
    }).then(function (r) {
      var data = r.data;
      data = data.result;
      data.forEach(function (v) {
        $('span[updated-id="' + v.updated_id + '"]').text(v.text);
      });
    });
  }
});
})();

/******/ })()
;