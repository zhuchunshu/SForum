/******/ (() => { // webpackBootstrap
/******/ 	var __webpack_modules__ = ({

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
/*!*******************************************!*\
  !*** ./resources/js/plugins/Core/user.js ***!
  \*******************************************/
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! axios */ "./node_modules/axios/index.js");
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(axios__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var izitoast__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! izitoast */ "./node_modules/izitoast/dist/js/iziToast.js");
/* harmony import */ var izitoast__WEBPACK_IMPORTED_MODULE_1___default = /*#__PURE__*/__webpack_require__.n(izitoast__WEBPACK_IMPORTED_MODULE_1__);



if (document.getElementById("vue-user-my-setting")) {
  var vums = {
    data: function data() {
      return {
        username: "",
        email: "",
        old_pwd: "",
        new_pwd: "",
        avatar: "",
        avatar_url: ""
      };
    },
    methods: {
      getFile: function getFile(event) {
        this.avatar = event.target.files[0];
      },
      submit: function submit(event) {
        event.preventDefault();
        axios__WEBPACK_IMPORTED_MODULE_0___default().post("/user/myUpdate", {
          _token: csrf_token,
          username: this.username,
          old_pwd: this.old_pwd,
          new_pwd: this.new_pwd,
          avatar: this.avatar
        }).then(function (r) {
          var data = r.data;
          console.log(data);
        });
      }
    },
    beforeMount: function beforeMount() {
      var _this = this;

      axios__WEBPACK_IMPORTED_MODULE_0___default().post("/user/data", {
        _token: csrf_token
      }).then(function (response) {
        _this.username = response.data.username;
        _this.email = response.data.email;
      })["catch"](function (error) {
        swal({
          title: "个人信息获取失败,详细查看控制台",
          icon: "error"
        });
        console.error(error);
      });
    }
  };
  Vue.createApp(vums).mount("#vue-user-my-setting");
} // 用户通知


$(function () {
  // action
  $('a[user-click="notice_action"]').click(function () {
    var href = $(this).attr("notice-href");
    var notice_id = $(this).attr("notice-id");
    axios__WEBPACK_IMPORTED_MODULE_0___default().post("/api/user/notice.read", {
      _token: csrf_token,
      notice_id: notice_id
    }).then(function (r) {
      location.href = href;
    });
  }); // 确认已读

  $('button[user-click="notice_read"]').click(function () {
    var notice_id = $(this).attr("notice-id");
    axios__WEBPACK_IMPORTED_MODULE_0___default().post("/api/user/notice.read", {
      _token: csrf_token,
      notice_id: notice_id
    }).then(function (r) {
      var data = r.data;

      if (data.success === false) {
        izitoast__WEBPACK_IMPORTED_MODULE_1___default().error({
          title: "Error",
          position: "topRight",
          message: data.result.msg
        });
      } else {
        izitoast__WEBPACK_IMPORTED_MODULE_1___default().success({
          title: "Success",
          position: "topRight",
          message: data.result.msg
        });
        setTimeout(function () {
          location.reload();
        }, 1500);
      }
    });
  });
});
$(function () {
  // 关注用户
  $('a[user-click="user_follow"]').click(function () {
    var user_id = $(this).attr("user-id");
    var th = $(this);
    axios__WEBPACK_IMPORTED_MODULE_0___default().post("/api/user/userfollow", {
      _token: csrf_token,
      user_id: user_id
    }).then(function (r) {
      var data = r.data;

      if (data.success === true) {
        if (data.code === 200) {
          th.children('span').text(data.result.msg);
        } else {
          th.children('span').text("关注");
        }

        izitoast__WEBPACK_IMPORTED_MODULE_1___default().success({
          title: "Success",
          message: data.result.msg,
          position: "topRight"
        });
      } else {
        izitoast__WEBPACK_IMPORTED_MODULE_1___default().error({
          title: "Error",
          message: data.result.msg,
          position: "topRight"
        });
      }
    })["catch"](function (e) {
      console.error(e);
      izitoast__WEBPACK_IMPORTED_MODULE_1___default().error({
        title: "Error",
        message: "请求出错,详细查看控制台",
        position: "topRight"
      });
    });
  }); //查询关注状态

  $('a[user-click="user_follow"]').each(function () {
    var user_id = $(this).attr("user-id");
    var th = $(this);
    axios__WEBPACK_IMPORTED_MODULE_0___default().post("/api/user/userfollow.data", {
      _token: csrf_token,
      user_id: user_id
    }).then(function (r) {
      var data = r.data;

      if (data.success === true) {
        th.children('span').text(data.result.msg);
      }
    })["catch"](function (e) {
      console.error(e);
      izitoast__WEBPACK_IMPORTED_MODULE_1___default().error({
        title: "Error",
        message: "请求出错,详细查看控制台",
        position: "topRight"
      });
    });
  });
});
})();

/******/ })()
;