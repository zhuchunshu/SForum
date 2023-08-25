import axios from "axios";
import swal from "sweetalert";
import iziToast from "izitoast";
import copy from "copy-to-clipboard";

window.onloadTurnstileCallback = function () {
    if (document.getElementById("captcha-container")) {
        turnstile.render('#captcha-container', {
            sitekey: captcha_config.cloudflare,
            theme: system_theme,
            callback: function (token) {
                console.log('Captcha token: ' + token)
                const captchaInputs = document.querySelectorAll('input[isCaptchaInput]');
                captchaInputs.forEach(input => {
                    input.value = token;
                });
                localStorage.setItem("captcha_token", token)
                const needCaptchaBtn = document.querySelectorAll('button[isNeedCaptcha]');
                needCaptchaBtn.forEach((button) => {
                    button.removeAttribute('disabled');
                });
            },
        });
    }
}

window.onloadGoogleRecaptchaCallback = function () {
    if (document.getElementById("captcha-container")) {
        grecaptcha.render('captcha-container', {
            'sitekey': captcha_config.recaptcha, //公钥
            'theme': system_theme, //主题颜色，有light与dark两个值可选
            'size': 'normal',//尺寸规则，有normal与compact两个值可选
            'callback': function (response) {
                console.log('Captcha token: ' + response)
                const captchaInputs = document.querySelectorAll('input[isCaptchaInput]');
                captchaInputs.forEach(input => {
                    input.value = response;
                });
                localStorage.setItem("captcha_token", response)
                const needCaptchaBtn = document.querySelectorAll('button[isNeedCaptcha]');
                needCaptchaBtn.forEach((button) => {
                    button.removeAttribute('disabled');
                });
            },

        })
    }
}

if (document.getElementById("vue-header-right-my")) {
    const vhrm = {
        data() {
            return {}
        },
        methods: {
            Logout() {
                axios
                    .post("/logout", {_token: csrf_token})
                    .then(function (response) {
                        var data = response.data;
                        if (data.success === false) {
                            swal({
                                title: "出错啦!",
                                text: data.result.msg,
                                icon: "error",
                            });
                        } else {
                            swal({
                                title: "Success!",
                                text: data.result.msg,
                                icon: "success",
                            });
                            setTimeout(() => {
                                location.href = data.result.url;
                            }, 1000);
                        }
                    })
                    .catch(function (error) {
                        swal("请求错误,详细查看控制台");
                        console.log(error);
                    });
            }
        }
    }
    Vue.createApp(vhrm).mount("#vue-header-right-my")
}

// 刷新菜单激活状态
$(function () {
    $("a[menu=active]").each(function () {
        const d = $(this);
        const _class = d.parent().parent().parent().parent().attr("class");
        d.parent().parent().parent().parent().attr('class', _class + " active")
    })
})

$(function () {
    $("img").each(function () {
        if ($(this).attr("data-src")) {
            var d = $(this)
            setTimeout(function () {
                d.attr("src", d.attr("data-src"))
            }, 500)
        }
    })
})

console.log("%cSForum %cwww.github.com/zhuchunshu/SForum", "color:#fff;background:linear-gradient(90deg,#448bff,#44e9ff);padding:5px 0;", "color:#000;background:linear-gradient(90deg,#44e9ff,#ffffff);padding:5px 10px 5px 0px;")


function get_user_config(){
    // 发起 POST 请求以获取用户配置
    axios.post("/api/user/get.user.config", {
        _token: csrf_token
    }).then(response => {
        const responseData = response.data;

        // 若请求不成功，则退出
        if (responseData.success === false) {
            return;
        }

        const userData = responseData.result;

        // 更新 UI 元素根据用户数据

        // 更新通知红点
        if (userData.notice_red > 0) {
            // 更新通知图标上的红点
            const notificationIcon = $("#core-notice-red");
            notificationIcon.show();
            notificationIcon.text(userData.notice_red);

            // 更新下拉菜单上的红点
            const dropdownRedDot = $("#common-user-notice-1");
            dropdownRedDot.show();
            dropdownRedDot.text(userData.notice_red);

            // 更新移动端红点
            const mobileRedDot = $("#common-user-notice-2");
            mobileRedDot.show();

            // 更新页头的样式
            const headerElement = $("div.border-primary");
            headerElement.addClass("border-orange");
            headerElement.removeClass("border-primary");
        }else{
            const notificationIcon = $("#core-notice-red");
            notificationIcon.hide();

            // 更新下拉菜单上的红点
            const dropdownRedDot = $("#common-user-notice-1");
            dropdownRedDot.hide();

            // 更新移动端红点
            const mobileRedDot = $("#common-user-notice-2");
            mobileRedDot.hide();

            // 更新页头的样式
            const headerElement = $("div.border-orange");
            headerElement.addClass("border-primary");
            headerElement.removeClass("border-orange");
        }
    });
}

get_user_config()

let timerId;

const startTimer = () => {
    timerId = setInterval(get_user_config, 5000);
};

const stopTimer = () => {
    clearInterval(timerId);
};

const handleVisibilityChange = () => {
    document.hidden ? stopTimer() : startTimer();
};

document.addEventListener("visibilitychange", handleVisibilityChange);
startTimer();



// if (ws_url && login_token){
//     var wsServer = ws_url+'/core?login-token='+login_token;
//     var websocket = new WebSocket(wsServer);
//     websocket.onopen = function (evt) {
//         //console.log("Connected to WebSocket server.");
//         //websocket.send('hello');
//     };
//
//     websocket.onclose = function (evt) {
//         //console.log("Disconnected");
//         $('span[core-show="online"]').remove();
//     };
//
//     websocket.onmessage = function (evt) {
//         var data = JSON.parse(evt.data);
//         $(function(){
//             $('span[core-show="online"]').each(function () {
//                 var user_id = $(this).attr("user-id");
//                 var y =$(this).attr("class")
//                 var title = $(this).attr("title")
//                 if(data.user.online.indexOf(user_id)!==-1){
//                     if(y!=="badge bg-success"){
//                         $(this).attr("class","badge bg-success")
//                     }
//                     if(title!=="在线"){
//                         $(this).attr("title","在线");
//                         $(this).attr("data-bs-original-title","在线");
//                         $(this).attr("aria-label","在线");
//                     }
//                 }else{
//                     if(y!=="badge bg-danger"){
//                         $(this).attr("class","badge bg-danger")
//                     }
//                     if(title!=="离线"){
//                         $(this).attr("title","离线");
//                         $(this).attr("data-bs-original-title","离线");
//                         $(this).attr("aria-label","离线");
//                     }
//                 }
//             })
//         })
//     };
//
//     websocket.onerror = function (evt, e) {
//         iziToast.error({
//             title: "Error",
//             message:"通信出错,详细查看控制台",
//             position:"topRight"
//         })
//         $('span[core-show="online"]').remove();
//         console.error('Error occured: ' + evt.data);
//     };
//
// }else{
//     $('span[core-show="online"]').remove();
// }

// 切换主题
$(function () {
    $("#core_update_theme").click(function () {
        let theme = $("body").attr("data-bs-theme")
        if (theme === "light") {
            $("body").attr("data-bs-theme", 'dark')
            $(this).html('<svg class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M0 0h24v24H0z" stroke="none"></path><circle cx="12" cy="12" r="4"></circle><path d="M3 12h1m8-9v1m8 8h1m-9 8v1M5.6 5.6l.7.7m12.1-.7l-.7.7m0 11.4l.7.7m-12.1-.7l-.7.7"></path></svg>');

        }
        if (theme === "dark") {
            $("body").attr("data-bs-theme", 'light')
            $(this).html('<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-moon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">\n' +
                '                    <desc>Download more icon variants from https://tabler-icons.io/i/moon</desc>\n' +
                '                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>\n' +
                '                    <path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z"></path>\n' +
                '                </svg>');
        }
        axios.post("/api/core/toggle.theme", {
            _token: csrf_token,
            theme: $("body").attr("data-bs-theme"),
        }).then(r => {
            let data = r.data;
            if (data.success === false) {
                iziToast.error({
                    title: "Error",
                    message: data.result.msg,
                    position: "topRight"
                })
            }
        })
    })
})


$(function () {
    $('a[name="core_update_theme"]').click(function () {
        let theme = $("body").attr("data-bs-theme")
        if (theme === "light") {
            $("body").attr("data-bs-theme", 'dark')
            $(this).html('<svg class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M0 0h24v24H0z" stroke="none"></path><circle cx="12" cy="12" r="4"></circle><path d="M3 12h1m8-9v1m8 8h1m-9 8v1M5.6 5.6l.7.7m12.1-.7l-.7.7m0 11.4l.7.7m-12.1-.7l-.7.7"></path></svg>');

        }
        if (theme === "dark") {
            $("body").attr("data-bs-theme", 'light')
            $(this).html('<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-moon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">\n' +
                '                    <desc>Download more icon variants from https://tabler-icons.io/i/moon</desc>\n' +
                '                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>\n' +
                '                    <path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z"></path>\n' +
                '                </svg>');
        }
        axios.post("/api/core/toggle.theme", {
            _token: csrf_token,
            theme: $("body").attr("data-bs-theme")
        }).then(r => {
            let data = r.data;
            if (data.success === false) {
                iziToast.error({
                    title: "Error",
                    message: data.result.msg,
                    position: "topRight"
                })
            }
        })
    })
})


$(function () {
    // 引用帖子
    $('a[core-click="copy"]').click(function () {
        const content = $(this).attr("copy-content")
        copy(content);
        var message = "复制成功";
        if ($(this).attr("message")) {
            message = $(this).attr("message")
        }
        iziToast.success({
            title: "Success",
            message: message,
            position: "topRight"
        })
    })
})


function GetQueryString(name) {
    const reg = eval("/" + name + "/g");
    const r = window.location.search.substr(1);
    const flag = reg.test(r);
    return !!flag;
}

if (GetQueryString('clean_topic_content_cache')) {
    localStorage.removeItem('topic_create_content');
    localStorage.removeItem('create_topic_title');
    localStorage.removeItem('create_topic_tag');
}
if (GetQueryString('clean_topic_comment_content_cache')) {
    localStorage.removeItem(getQueryVariable('clean_topic_comment_content_cache'));
    history.pushState('', '', urlDel('clean_topic_comment_content_cache'))
}

function urlDel(name) {
    var url = window.location;
    var baseUrl = url.origin + url.pathname + "?";
    var query = url.search.substr(1);
    if (query.indexOf(name) > -1) {
        var obj = {}
        var arr = query.split("&");
        for (var i = 0; i < arr.length; i++) {
            arr[i] = arr[i].split("=");
            obj[arr[i][0]] = arr[i][1];
        }
        ;
        delete obj[name];
        var url = baseUrl + JSON.stringify(obj).replace(/[\"\{\}]/g, "").replace(/\:/g, "=").replace(/\,/g, "&");
        return url
    } else {
        return window.location.href;
    }
    ;
}

function getQueryVariable(variable) {
    var query = window.location.search.substring(1);
    var vars = query.split("&");
    for (var i = 0; i < vars.length; i++) {
        var pair = vars[i].split("=");
        if (pair[0] === variable) {
            return pair[1];
        }
    }
    return false;
}

$(function () {
    if (theme_status === false && matchMedia('(prefers-color-scheme: dark)').matches !== auto_theme) {
        if (matchMedia('(prefers-color-scheme: dark)').matches) {
            $("body").attr("data-bs-theme", 'dark')
        } else {
            $("body").attr("data-bs-theme", 'light')
        }
        axios.post("/api/core/toggle.auto.theme", {
            _token: csrf_token,
            theme: $("body").attr("data-bs-theme")
        })
    }
})

// 处理暗黑模式状态变化的函数
function handleDarkModeChange() {
    $(function () {
        if (theme_status === false && matchMedia('(prefers-color-scheme: dark)').matches !== auto_theme) {
            if (matchMedia('(prefers-color-scheme: dark)').matches) {
                $("body").attr("data-bs-theme", 'dark')
            } else {
                $("body").attr("data-bs-theme", 'light')
            }
            axios.post("/api/core/toggle.auto.theme", {
                _token: csrf_token,
                theme: $("body").attr("data-bs-theme")
            })
        }
    })
}

// 添加事件监听器到媒体查询
const darkModeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
darkModeMediaQuery.addListener(handleDarkModeChange);

// 自动刷新验证码
document.addEventListener("DOMContentLoaded", function() {
    const buttons = document.querySelectorAll("button[isNeedCaptcha]");

    buttons.forEach(button => {
        button.addEventListener("click", function() {
            switch (captcha_config.service) {
                case "google":
                    grecaptcha.reset();
                    break;
                case "cloudflare":
                    turnstile.reset();
                    break;
            }
        });
    });
});