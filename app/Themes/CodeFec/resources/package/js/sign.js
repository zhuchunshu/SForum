import axios from "axios";
import swal from "sweetalert";

if(document.getElementById("vue-core-sign-register")){
    const vcsr = {
        data() {
            return {
                username:null,
                email:null,
                password:null,
                cfpassword:null,
                captcha:null,
                invitationCode:null,
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
                    invitationCode:this.invitationCode
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
            }
        },
        mounted() {

        }
    }

    Vue.createApp(vcsr).mount("#vue-core-sign-register")
}


// 邮箱登陆
if(document.getElementById("vue-core-sign-login")){
    const vcsl = {
        data(){
            return {
                email:null,
                password:null,
                captcha:null
            }
        },
        methods:{
            submit(){
                axios.post("/login",{
                    _token:csrf_token,
                    email:this.email,
                    password:this.password,
                    captcha:this.captcha,
                })
                    .then(response=>{
                        var data = response.data;
                        if(data.success===true){
                            swal({
                                icon:"success",
                                title:data.result.msg
                            })
                            setTimeout(()=>{
                                location.href="/";
                            },1000)

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

// 邮箱登陆
if(document.getElementById("vue-core-sign-login-username")){
    const app = {
        data(){
            return {
                username:null,
                password:null,
                captcha:null
            }
        },
        methods:{
            submit(){
                axios.post("/login/username",{
                    _token:csrf_token,
                    username:this.username,
                    password:this.password,
                    captcha:this.captcha,
                })
                    .then(response=>{
                        var data = response.data;
                        if(data.success===true){
                            swal({
                                icon:"success",
                                title:data.result.msg
                            })
                            setTimeout(()=>{
                                location.href="/";
                            },1000)

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

    Vue.createApp(app).mount("#vue-core-sign-login-username")
}


// 找回密码
if(document.getElementById("vue-core-forgot-password")){
    const app = {
        data(){
            return {
                email:null,
                captcha:null,
                show:true,
                YzCode:null,
                sendCoded:true,
                setpwd:true,
                setPwd_password:null,
                setPwd_cfpassword:null,
                setPwd_submit:false,
                password:null,
            }
        },
        methods:{
            sendCode(){
                axios.post("/forgot-password/sendCode",{
                    _token:csrf_token,
                    email:this.email,
                    captcha:this.captcha,
                })
                    .then(response=>{
                        var data = response.data;
                        if(data.success===true){
                            swal({
                                icon:"success",
                                title:data.result.msg
                            })
                            this.show=!this.show;
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
            submit(){
                axios.post("/forgot-password/verifyCode",{
                    _token:csrf_token,
                    code:this.YzCode,
                })
                    .then(response=>{
                        var data = response.data;
                        if(data.success===true){
                            swal({
                                icon:"success",
                                title:data.result.msg
                            })
                            this.sendCoded=!this.sendCoded;
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
            // 设置新密码
            setPwdSubmit(){
                if(this.setPwd_password!==this.setPwd_cfpassword){
                    swal({
                        title:"两次输入密码不一致",
                        icon:"error"
                    })
                }
                axios.post("/forgot-password",{
                    _token:csrf_token,
                    cfpassword:this.setPwd_cfpassword,
                    password:this.setPwd_password,
                    code:this.YzCode,
                    email:this.email
                })
                    .then(response=>{
                        var data = response.data;
                        if(data.success===true){
                            swal({
                                icon:"success",
                                title:data.result.msg
                            })
                            this.setpwd=false;

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
            }
        }
    }

    Vue.createApp(app).mount("#vue-core-forgot-password")
}