/******/ (() => { // webpackBootstrap
/******/ 	var __webpack_modules__ = ({

/***/ "./app/Plugins/User/resources/package/js/pm.js":
/*!*****************************************************!*\
  !*** ./app/Plugins/User/resources/package/js/pm.js ***!
  \*****************************************************/
/***/ ((module) => {

function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

function _defineProperties(target, props) { for (var i = 0; i < props.length; i++) { var descriptor = props[i]; descriptor.enumerable = descriptor.enumerable || false; descriptor.configurable = true; if ("value" in descriptor) descriptor.writable = true; Object.defineProperty(target, descriptor.key, descriptor); } }

function _createClass(Constructor, protoProps, staticProps) { if (protoProps) _defineProperties(Constructor.prototype, protoProps); if (staticProps) _defineProperties(Constructor, staticProps); return Constructor; }

// 私信
var OwO = /*#__PURE__*/function () {
  function OwO(option) {
    var _this = this;

    _classCallCheck(this, OwO);

    var defaultOption = {
      logo: 'OwO表情',
      container: document.getElementsByClassName('OwO')[0],
      target: document.getElementsByTagName('textarea')[0],
      position: 'down',
      width: '100%',
      maxHeight: '250px',
      api: 'https://api.anotherhome.net/OwO/OwO.json'
    };

    for (var defaultKey in defaultOption) {
      if (defaultOption.hasOwnProperty(defaultKey) && !option.hasOwnProperty(defaultKey)) {
        option[defaultKey] = defaultOption[defaultKey];
      }
    }

    this.container = option.container;
    this.target = option.target;

    if (option.position === 'up') {
      this.container.classList.add('OwO-up');
    }

    var xhr = new XMLHttpRequest();

    xhr.onreadystatechange = function () {
      if (xhr.readyState === 4) {
        if (xhr.status >= 200 && xhr.status < 300 || xhr.status === 304) {
          _this.odata = JSON.parse(xhr.responseText);

          _this.init(option);
        } else {
          console.log('OwO data request was unsuccessful: ' + xhr.status);
        }
      }
    };

    xhr.open('get', option.api, true);
    xhr.send(null);
  }

  _createClass(OwO, [{
    key: "init",
    value: function init(option) {
      var _this2 = this;

      this.area = option.target;
      this.packages = Object.keys(this.odata); // fill in HTML

      var html = "\n            <div class=\"OwO-logo\"><span>".concat(option.logo, "</span></div>\n            <div class=\"OwO-body\" style=\"width: ").concat(option.width, "\">");

      for (var i = 0; i < this.packages.length; i++) {
        html += "\n                <ul class=\"OwO-items OwO-items-".concat(this.odata[this.packages[i]].type, "\" style=\"max-height: ").concat(parseInt(option.maxHeight) - 53 + 'px', ";\">");
        var opackage = this.odata[this.packages[i]].container;

        for (var _i = 0; _i < opackage.length; _i++) {
          html += "\n                    <li class=\"OwO-item\" title=\"".concat(opackage[_i].text, "\">").concat(opackage[_i].icon, "</li>");
        }

        html += "\n                </ul>";
      }

      html += "\n                <div class=\"OwO-bar\">\n                    <ul class=\"OwO-packages\">";

      for (var _i2 = 0; _i2 < this.packages.length; _i2++) {
        html += "\n                        <li><span>".concat(this.packages[_i2], "</span></li>");
      }

      html += "\n                    </ul>\n                </div>\n            </div>\n            ";
      this.container.innerHTML = html; // bind event

      this.logo = this.container.getElementsByClassName('OwO-logo')[0];
      this.logo.addEventListener('click', function () {
        _this2.toggle();
      });
      this.container.getElementsByClassName('OwO-body')[0].addEventListener('click', function (e) {
        var target = null;

        if (e.target.classList.contains('OwO-item')) {
          target = e.target;
        } else if (e.target.parentNode.classList.contains('OwO-item')) {
          target = e.target.parentNode;
        }

        if (target) {
          var cursorPos = _this2.area.selectionEnd;
          var areaValue = _this2.area.value;
          var tag = e.target.getElementsByTagName('img');

          if (e.target.nodeName !== "IMG") {
            if (tag.length > 0) {
              _this2.area.value = areaValue.slice(0, cursorPos) + " " + e.target.title + " " + areaValue.slice(cursorPos);
            } else {
              _this2.area.value = areaValue.slice(0, cursorPos) + " " + target.innerHTML + " " + areaValue.slice(cursorPos);
            }
          } else {
            _this2.area.value = areaValue.slice(0, cursorPos) + " " + e.target.parentElement.title + " " + areaValue.slice(cursorPos);
          }

          _this2.area.focus();

          _this2.toggle();
        }
      });
      this.packagesEle = this.container.getElementsByClassName('OwO-packages')[0];

      var _loop = function _loop(_i3) {
        (function (index) {
          _this2.packagesEle.children[_i3].addEventListener('click', function () {
            _this2.tab(index);
          });
        })(_i3);
      };

      for (var _i3 = 0; _i3 < this.packagesEle.children.length; _i3++) {
        _loop(_i3);
      }

      this.tab(0);
    }
  }, {
    key: "toggle",
    value: function toggle() {
      if (this.container.classList.contains('OwO-open')) {
        this.container.classList.remove('OwO-open');
      } else {
        this.container.classList.add('OwO-open');
      }
    }
  }, {
    key: "tab",
    value: function tab(index) {
      var itemsShow = this.container.getElementsByClassName('OwO-items-show')[0];

      if (itemsShow) {
        itemsShow.classList.remove('OwO-items-show');
      }

      this.container.getElementsByClassName('OwO-items')[index].classList.add('OwO-items-show');
      var packageActive = this.container.getElementsByClassName('OwO-package-active')[0];

      if (packageActive) {
        packageActive.classList.remove('OwO-package-active');
      }

      this.packagesEle.getElementsByTagName('li')[index].classList.add('OwO-package-active');
    }
  }]);

  return OwO;
}();

if ( true && typeof module.exports !== 'undefined') {
  module.exports = OwO;
} else {
  window.OwO = OwO;
} // pm


if (document.getElementById('user-pm-container')) {
  var app = {
    data: function data() {
      return {
        socket: null,
        msg: null,
        to_id: to_id,
        btn: {
          disabled: false
        },
        messages: 0
      };
    },
    mounted: function mounted() {
      var _this3 = this;

      this.InitOwO();
      this.InitSocket();
      setInterval(function () {
        _this3.msg = document.getElementsByTagName('textarea')[0].value;
      }, 300);
    },
    methods: {
      InitOwO: function InitOwO() {
        new OwO({
          logo: '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-mood-smile" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">\n' + '                                                <desc>Download more icon variants from https://tabler-icons.io/i/mood-smile</desc>\n' + '                                                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>\n' + '                                                <circle cx="12" cy="12" r="9"></circle>\n' + '                                                <line x1="9" y1="10" x2="9.01" y2="10"></line>\n' + '                                                <line x1="15" y1="10" x2="15.01" y2="10"></line>\n' + '                                                <path d="M9.5 15a3.5 3.5 0 0 0 5 0"></path>\n' + '                                            </svg>',
          container: document.getElementsByClassName('OwO')[0],
          target: document.getElementsByClassName('OwO-textarea')[0],
          api: '/api/core/OwO.json',
          position: 'down',
          width: '100%',
          maxHeight: '250px'
        });
      },
      // 初始化socket
      InitSocket: function InitSocket() {
        var _this4 = this;

        this.socket = io(pm_socket, {
          transports: ["websocket"]
        });
        this.socket.on("connect", function () {
          if (_this4.socket.connected === false) {
            swal({
              title: "聊天室连接失败!",
              icon: "error"
            });
            return;
          }

          _this4.socket.emit('join-room', '{"token":"' + _token + '","to_id":"' + to_id + '"}');

          setInterval(function () {
            _this4.socket.emit('getMsg', '{"token":"' + _token + '","to_id":"' + to_id + '"}');

            _this4.socket.on('getMsg', function (data) {
              _this4.messages = data;
            });
          }, 3000);
        });
      },
      // 发消息
      sendMsg: function sendMsg() {
        console.log(this.msg);

        if (!this.msg) {
          swal({
            title: "不能发送空消息",
            icon: "error"
          });
          return;
        }

        this.btn.disabled = this.btn.disabled === false;

        if (this.socket.emit('sendMsg', '{"token":"' + _token + '","to_id":"' + to_id + '","msg" : "' + this.msg + '"}')) {
          this.btn.disabled = this.btn.disabled === false;
          this.msg = null;
          location.reload();
        }
      }
    }
  };
  Vue.createApp(app).mount('#user-pm-container');
}

$('#chat-list').scrollTop($("#chat-list")[0].scrollHeight);

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
/******/ 		__webpack_modules__[moduleId](module, module.exports, __webpack_require__);
/******/ 	
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/ 	
/************************************************************************/
/******/ 	
/******/ 	// startup
/******/ 	// Load entry module and return exports
/******/ 	// This entry module is referenced by other modules so it can't be inlined
/******/ 	var __webpack_exports__ = __webpack_require__("./app/Plugins/User/resources/package/js/pm.js");
/******/ 	
/******/ })()
;