import axios from "axios"
import swal from 'sweetalert';

function getCaptchaInputValue(){
    const inputs = document.querySelectorAll('input[isCaptchaInput]');
    let v;
    if(inputs.length>0){
        v = inputs[0].value
    }
    if(!v){
        v = localStorage.getItem("captcha_token")
    }
    return v;
}

const form = {
    data() {
        return {
            username:null,
            password:null,
            csrf_token:null,
            captcha:null
        }
    },
    methods: {
        submit(){
            if(getCaptchaInputValue()){
                this.captcha = getCaptchaInputValue()
            }
            axios.post("/admin/login",{username:this.username,password:this.password,_token:csrf_token,captcha:this.captcha})
            .then(function(response){
                var data = response.data
                var content="";
                if(data.success===false){
                    if(data.result instanceof Array){
                        data.result.forEach(element => {
                            content = content+element+"\n"
                        });
                        swal({
                            icon : "error",
                            title: "出错啦!",
                            text:content
                        });
                    }
                }else{
                    // 登陆成功
                    swal({
                        icon : "success",
                        title: "Success",
                        text:   data.result.msg
                    });
                    setTimeout(() => {
                        location.href=data.result.url
                    }, 1500);
                }
            })
            .catch(function(error){
                swal({
                    icon : "error",
                    title: "请求错误,详细查看控制台输出",
                });
                console.log(error)
            })
        }
    },
}
Vue.createApp(form).mount("#form")

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