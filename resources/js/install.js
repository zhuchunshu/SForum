import axios from "axios";
var qs = require('querystring')
import swal from "sweetalert"

if(document.getElementById("app-install")){

    const app = {
        data(){
            return {
                progress:25,
                step:1,
                tips:null,
                get_install_lock:false,
                install_lock:0,
                env:null,
                email:null,
                username:null,
                password:null,
            }
        },
        methods:{
            load(){
                axios.post("/install",{
                    _token:csrf_token
                }).then(r=>{
                    const data = r.data;
                    this.progress = data.progress;
                    this.step = data.step;
                    this.tips = data.tips;
                    this.install_lock = data.install_lock;
                    this.get_install_lock = true;
                })
                axios.post("/install/env",{
                    _token:csrf_token
                }).then(r=>{
                    console.log(r.data)
                    this.env=r.data
                })
            },
            // 下一步
            next(){
                axios.post("/install/next",{
                    _token:csrf_token,
                    env:qs.stringify(this.env)
                }).then(r=>{
                    const data  = r.data;
                   if(data.success){
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

                // 重载页面
                this.load()
            },
            // 上一步
            previous(){
                axios.post("/install/previous",{
                    _token:csrf_token
                })

                // 重载页面
                this.load()
            },
            install(){
                if(!this.email){
                    swal({
                        title:"请填写邮箱!",
                        icon:'error'
                    })
                }
                if(!this.username){
                    swal({
                        title:"请填写用户名!",
                        icon:'error'
                    })
                }
                if(!this.password){
                    swal({
                        title:"请填写密码!",
                        icon:'error'
                    })
                }
                axios.post("/install/next",{
                    _token:csrf_token,
                    username:this.username,
                    email:this.email,
                    password:this.password
                }).then(r=>{
                    const data  = r.data;
                    if(data.success){
                        swal({
                            title:data.result.msg,
                            icon:'success'
                        })
                        setTimeout(()=>{
                            location.href="/admin"
                        },1500)
                    }else{
                        swal({
                            title:data.result.msg,
                            icon:'error'
                        })
                    }
                })
            }
        },
        mounted(){
            this.load()
        }
    }

    Vue.createApp(app).mount("#app-install")

}
