import axios from "axios";
import swal from "sweetalert";

if(document.getElementById("vue-core-sign-register")){
    const vcsr = {
        data() {
            return {
                username:"",
                email:"",
                password:"",
                cfpassword:"",
                captcha:"",
                endTime:CaptchaEndTime
            }
        },
        methods: {
            submit(){
                axios.post("/register",{
                    _token:csrf_token,
                    username:this.username,
                    email:this.email,
                    password:this.password,
                    cfpassword:this.cfpassword,
                    captcha:this.captcha,
                    timer:""
                })
                    .then(response=>{
                        var data = response.data;
                        if(data.success===true){
                            swal({
                                icon:"success",
                                title:data.result.msg
                            })
                            setTimeout(() => {
                                location.href="/login"
                            }, 1200);
                        }else{
                            if(data.result instanceof Array){
                                var content = "";
                                data.result.forEach(element => {
                                    content = content+element+"\n"
                                });
                                swal({
                                    icon : "error",
                                    title: "出错啦!",
                                    text:content
                                });
                            }else{
                                swal({
                                    icon:"error",
                                    title:data.result.msg
                                })
                            }

                        }
                    })
                    .catch(error=>{
                        console.error(error)
                        swal({
                            title:"请求出错",
                            icon:"error"
                        })
                    })
            },
            setTimeOut(){
                this.endTime --;
            }
        },
        mounted() {
            this.timer = setInterval(this.setTimeOut, 1000);
        }
    }

    Vue.createApp(vcsr).mount("#vue-core-sign-register")
}


if(document.getElementById("vue-core-sign-login")){
    const vcsl = {
        data(){
            return {
                email:"",
                password:""
            }
        },
        methods:{
            submit(){
                axios.post("/login",{
                    _token:csrf_token,
                    email:this.email,
                    password:this.password,
                })
                    .then(response=>{
                        var data = response.data;
                        if(data.success===true){
                            swal({
                                icon:"success",
                                title:data.result.msg
                            })
                            setTimeout(() => {
                                location.href="/"
                            }, 1200);
                        }else{
                            if(data.result instanceof Array){
                                var content = "";
                                data.result.forEach(element => {
                                    content = content+element+"\n"
                                });
                                swal({
                                    icon : "error",
                                    title: "出错啦!",
                                    text:content
                                });
                            }else{
                                swal({
                                    icon:"error",
                                    title:data.result.msg
                                })
                            }

                        }
                    })
                    .catch(error=>{
                        console.error(error)
                        swal({
                            title:"请求出错",
                            icon:"error"
                        })
                    })
            },
        }
    }

    Vue.createApp(vcsl).mount("#vue-core-sign-login")
}