import axios from "axios"
import swal from "sweetalert"
var qs = require('querystring')

if(document.getElementById("vue-im-form")){
    const vue_im_form = {
        data() {
            return {
                old_pwd:"",
                new_pwd:"",
                check_username:true,
                check_email:false,
                check_password:false,
                username:admin.username,
                email:admin.email,
            }
        },
        methods: {
            submit(){
                if(this.check_username){
                    // 修改用户名
                    axios.post("/admin/setting/im",{
                        type:'username',
                        username:this.username,
                        _token:csrf_token
                    })
                        .then(function(response){
                            var data = response.data;
                            if(data.success===false){
                                swal({
                                    icon:"error",
                                    title:data.result.msg
                                })
                            }else{
                                swal({
                                    icon:"success",
                                    title:data.result.msg
                                })
                            }
                        })
                        .catch(function(error){
                            console.log(error)
                            swal({
                                title:"用户名修改失败,详细查看控制台",
                                icon:"error"
                            })
                        })
                }
                if(this.check_email){
                    // 修改邮箱
                    axios.post("/admin/setting/im",{
                        type:'email',
                        email:this.email,
                        _token:csrf_token
                    })
                        .then(function(response){
                            var data = response.data;
                            if(data.success===false){
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
                                    icon:"success",
                                    title:data.result.msg
                                })
                            }
                        })
                        .catch(function(error){
                            console.log(error)
                            swal({
                                title:"用户名修改失败,详细查看控制台",
                                icon:"error"
                            })
                        })
                }
                if(this.check_password){
                    // 修改密码
                    axios.post("/admin/setting/im",{
                        type:'password',
                        old_pwd:this.old_pwd,
                        new_pwd:this.new_pwd,
                        _token:csrf_token
                    })
                        .then(function(response){
                            var data = response.data;
                            if(data.success===false){
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
                                    icon:"success",
                                    title:data.result.msg
                                })
                            }
                        })
                        .catch(function(error){
                            console.log(error)
                            swal({
                                title:"用户名修改失败,详细查看控制台",
                                icon:"error"
                            })
                        })
                }
            }
        },
    }

    Vue.createApp(vue_im_form).mount("#vue-im-form")
}



// 站点设置 --core
if(document.getElementById("setting-core-form")){
    const scf = {
        data() {
            return {
                data:{}
            }
        },
        beforeMount() {
            axios.post("/api/adminOptionList",{_token:csrf_token})
            .then(response=>{
                var data = response.data;
                if(data.success===true){
                    this.data=data.result;
                }else{
                    swal({
                        title:data.result.msg,
                        icon:'error'
                    })
                }
            })
            .catch(error=>{
                console.error(error)
                swal({
                    title:"请求出错,详细请查看控制台",
                    icon:"error"
                })
            })
        },
        methods: {
            submit(){
                axios.post("/admin/setting",{_token:csrf_token,data:qs.stringify(this.data)})
                .then(response=>{
                    var data = response.data;
                    if(data.success===true){
                        swal({
                            title:data.result.msg,
                            icon:'success'
                        })
                    }else{
                        swal({
                            title:data.result.msg,
                            icon:'error'
                        })
                    }
                })
                .catch(error=>{
                    console.error(error)
                    swal({
                        title:"请求出错,详细请查看控制台",
                        icon:"error"
                    })
                })
            }
        }
    }
    Vue.createApp(scf).mount("#setting-core-form")
}