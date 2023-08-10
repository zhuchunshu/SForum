import axios from "axios";
import iziToast from "izitoast";
import qs from "querystring";
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

if (document.getElementById('user-order-show-paying')) {
    const app = {
        data() {
            return {
                order_id: order_id,
                payment: null,
                qrcode_url: null,
                pay_url: null,
                paid: false,
            }
        },
        methods: {
            submit() {
                this.qrcode_url = null;
                if (!this.payment) {
                    swal('出错啦', '请选择一种支付方式', 'error')
                    return;
                }
                // 获取付款链接
                axios.post('/user/order/' + this.order_id + '.order/paying', {
                    _token: csrf_token,
                    payment: this.payment
                }).then(r => {
                    let data = r.data;
                    if (data.success === false) {
                        iziToast.error({
                            title: "error",
                            message: data.result.msg,
                            position: "topRight",
                            timeout: 10000
                        })
                        return;
                    }
                    this.qrcode_url = "/api/core/qr_code?content=" + data.result.url
                    this.pay_url = data.result.url
                    this.checkPaid()
                })
            },
            checkPaid() {
                setInterval(() => {
                    axios.post('/user/order/' + this.order_id + '.order/status', {
                        _token: csrf_token
                    }).then(r => {
                        let data = r.data
                        if (data.success === false) {
                            console.error(data)
                        } else {
                            if (data.result.status === "支付成功") {
                                this.paid = true;
                                setTimeout(() => {
                                    location.reload();
                                }, 1500)
                            }
                        }
                    })
                }, 2000)
            }
        }
    }
    Vue.createApp(app).mount("#user-order-show-paying")
}

if (document.getElementById("user-data-money-recharge")) {
    const app = {
        data() {
            return {
                payment: null,
                amount: null,
                qrcode_url: null,
                captcha: null,
                pay_url: null,
                paid: false,
                order_id: null,
                btn_disabled: false
            }
        },
        methods: {
            submit() {
                if(getCaptchaInputValue()){
                    this.captcha = getCaptchaInputValue()
                }
                if (!this.payment) {
                    swal('Error', '请选择支付方式', 'error')
                    return;
                }
                if (!this.amount) {
                    swal('Error', '充值金额不能为空', 'error')
                    return;
                }
                if (!this.captcha) {
                    swal('Error', '请输入验证码', 'error')
                    return;
                }
                this.btn_disabled = true;
                axios.post('/user/asset/money.recharge', {
                    _token: csrf_token,
                    amount: this.amount,
                    payment: this.payment,
                    captcha: this.captcha
                }).then(r => {
                    const data = r.data;
                    this.btn_disabled = false;
                    if (data.success === false) {
                        swal('请求出错!', data.result.msg, 'error')
                        return;
                    }
                    this.qrcode_url = "/api/core/qr_code?content=" + data.result.url
                    this.pay_url = data.result.url
                    this.order_id = data.result.order_id
                    this.checkPaid()
                })
            },
            checkPaid() {
                setInterval(() => {
                    axios.post('/user/order/' + this.order_id + '.order/status', {
                        _token: csrf_token
                    }).then(r => {
                        let data = r.data
                        if (data.success === false) {
                            console.error(data)
                        } else {
                            if (data.result.status === "支付成功") {
                                this.paid = true;
                                setTimeout(() => {
                                    location.href = "/user/order/" + this.order_id + ".order";
                                }, 1500)
                            }
                        }
                    })
                }, 2000)
            }
        }
    }
    Vue.createApp(app).mount('#user-data-money-recharge')
}

if (document.getElementById('user-data-exchange')) {
    const app = {
        data() {
            return {
                data: {},
                selected: user_exchange_selected,
                submit_urls: submit_urls,
                captcha:null
            }
        },
        mounted() {

        },
        methods: {
            submit() {
                if(getCaptchaInputValue()){
                    this.captcha = getCaptchaInputValue()
                }
                if(!this.captcha){
                    swal('Error','请输入验证码','error')
                    return ;
                }
                axios.post(this.submit_urls[this.selected], {
                    _token: csrf_token,
                    data: qs.stringify(this.data),
                    captcha:this.captcha
                }).then(r => {
                    const data = r.data;
                    if(data.success===false){
                        swal('Error',data.result.msg,'error')
                    }else{
                        if(data.code===201){
                            setTimeout(()=>{
                                location.reload()
                            },2000)
                        }
                        swal('Success',data.result.msg+"",'success')
                    }
                })
                //console.log(qs.stringify(this.data))
            }
        },
        watch: {
            selected(val) {
                console.log(val)
            }
        }
    }
    Vue.createApp(app).mount('#user-data-exchange')
}