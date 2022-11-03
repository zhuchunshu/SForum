import axios from "axios";
import swal from "sweetalert";
import iziToast from "izitoast";
import download from "downloadjs";

if(document.getElementById("vue-user-create-class")){
    const vucc = {
        data(){
            return {
                name:"",
                icon:"",
                color:"#206bc4",
                quanxian:[],
                permission_value : 1
            }
        },
        mounted(){
          axios.post("/admin/userClass/Default/Authority",{_token:csrf_token})
              .then(r=>{
                  this.quanxian = r.data
              })
        },
        methods:{
            submit(){
                axios.post("/admin/userClass/create",{
                    _token:csrf_token,
                    name:this.name,
                    icon:this.icon,
                    color:this.color,
                    quanxian:this.quanxian,
                    'permission-value':this.permission_value
                })
                .then(response=>{
                    var data = response.data;
                    if(data.success===true){
                        swal({
                            icon:"success",
                            title:data.result.msg
                        })
                        setTimeout(() => {
                            location.href="/admin/userClass"
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
                    swal({
                        title:"请求出错,详细查看控制台",
                        icon:"error"
                    })
                    console.error(error)
                })
            }
        }
    }
    Vue.createApp(vucc).mount("#vue-user-create-class")
}

if(document.getElementById("vue-user-edit-class")){
    const vsec = {
        data(){
            return {
                id:userClassId,
                name:"",
                icon:"",
                color:"#206bc4",
                quanxian:[],
                permission_value:1
            }
        },
        beforeMount() {
            axios.post("/admin/userClass/"+this.id+"/data",{_token:csrf_token})
            .then(response=>{
                var data = response.data;
                if(data.success===false){
                    swal({
                        icon:"error",
                        title:data.result.msg
                    })
                }else{
                    var result = data.result;
                    this.name = result.name;
                    this.icon = result.icon;
                    this.color = result.color;
                    this.quanxian = result.quanxian;
                    this.permission_value = result.permission_value;
                }

            })
            .catch(error=>{
                console.error(error)
            })
        },
        methods:{
            submit(){
                axios.post("/admin/userClass/update",{
                    id:this.id,
                    _token:csrf_token,
                    name:this.name,
                    icon:this.icon,
                    color:this.color,
                    quanxian:this.quanxian,
                    'permission-value':this.permission_value
                })
                .then(response=>{
                    var data = response.data;
                    if(data.success===true){
                        swal({
                            icon:"success",
                            title:data.result.msg
                        })
                        setTimeout(() => {
                            location.href="/admin/userClass"
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
                    swal({
                        title:"请求出错,详细查看控制台",
                        icon:"error"
                    })
                    console.error(error)
                })
            }
        }
    }
    Vue.createApp(vsec).mount("#vue-user-edit-class");
}


if(document.getElementById("vue-users")){
    const vue_users = {
        data() {
            return {

            }
        },
        methods:{
            username(id,username){
                swal({
                    title: "修改ID为:" + id + "的用户名",
                    content: {
                        element: "input",
                        attributes: {
                            placeholder: "请输入新的用户名",
                            type: "text",
                            value: username
                        },
                    },
                }).then((r)=>{
                    if(!r){
                        return ;
                    }
                    if(r===username){
                        swal({
                            icon:"error",
                            title:"用户名未发生改变"
                        })
                        return ;
                    }
                    axios.post("/admin/users/update/username",{
                        _token:csrf_token,
                        username:r,
                        user_id:id
                    }).then((r)=>{
                        const data = r.data;
                        if(!data.success){
                            swal({
                                title:data.result.msg,
                                icon:"error",
                            })
                        }else{
                            swal({
                                title:data.result.msg,
                                icon:"success",
                            })
                        }
                    }).catch(e=>{
                        console.error(e)
                        swal({
                            title:"请求出错,详细查看控制台",
                            icon:"error"
                        })
                    })
                });
            },
            email(id,email){
                swal({
                    title: "修改ID为:" + id + "的邮箱",
                    content: {
                        element: "input",
                        attributes: {
                            placeholder: "请输入新的邮箱账号",
                            type: "email",
                            value: email
                        },
                    },
                }).then((r)=>{
                    if(!r){
                        return ;
                    }
                    if(r===email){
                        swal({
                            icon:"error",
                            title:"邮箱未发生改变"
                        })
                        return ;
                    }
                    axios.post("/admin/users/update/email",{
                        _token:csrf_token,
                        email:r,
                        user_id:id
                    }).then((r)=>{
                        const data = r.data;
                        if(!data.success){
                            swal({
                                title:data.result.msg,
                                icon:"error",
                            })
                        }else{
                            swal({
                                title:data.result.msg,
                                icon:"success",
                            })
                        }
                    }).catch(e=>{
                        console.error(e)
                        swal({
                            title:"请求出错,详细查看控制台",
                            icon:"error"
                        })
                    })
                });
            },
            UserClass(id,class_id){
                location.href="/admin/users/update/"+id+"/UserClass"
            },
            token(id,token){
                swal({
                    title:"确定要更新此用户的Token吗?",
                    buttons: ["取消", "确定"],
                }).then(r=>{
                    if(r===true){
                        axios.post("/admin/users/update/token", {
                            _token: csrf_token,
                            user_id: id
                        }).then(r  =>{
                            const data = r.data;
                            if(!data.success){
                                swal({
                                    title:data.result.msg,
                                    icon:"error",
                                })
                            }else{
                                swal({
                                    title:data.result.msg,
                                    icon:"success",
                                })
                            }
                        }).catch(e=>{
                            console.error(e)
                            swal({
                                title:"请求出错,详细查看控制台",
                                icon:"error"
                            })
                        })
                    }
                })
            },
            re_pwd(id){
                swal("将密码修改为什么?", {
                    content: {
                        element: "input",
                        attributes: {
                            placeholder: "请输入新的密码",
                            type: "password",
                        },
                    },
                })
                    .then((value) => {
                        if(value){
                            axios.post("/admin/users/update/password", {
                                _token:csrf_token,
                                user_id:id,
                                password:value
                            }).then(r=>{
                                const data = r.data;
                                if(!data.success){
                                    swal({
                                        title:data.result.msg,
                                        icon:"error",
                                    })
                                }else{
                                    swal({
                                        title:data.result.msg,
                                        icon:"success",
                                    })
                                }
                            }).catch(e=>{
                                console.error(e)
                                swal({
                                    title:"请求出错,详细查看控制台",
                                    icon:"error"
                                })
                            })
                        }
                    });
            },
            remove(id){
                swal({
                    title:"确定要删除此用户吗? 删除后不可恢复!!!",
                    buttons: ["取消", "确定"],
                }).then(r=>{
                    if(r===true){
                        axios.post("/admin/users/remove", {
                            _token: csrf_token,
                            user_id: id
                        }).then(r  =>{
                            const data = r.data;
                            if(!data.success){
                                swal({
                                    title:data.result.msg,
                                    icon:"error",
                                }).then(r=>{
                                    location.reload()
                                })
                                setTimeout(function(){
                                    location.reload()
                                },1200)
                            }else{
                                swal({
                                    title:data.result.msg,
                                    icon:"success",
                                }).then(r=>{
                                    location.reload()
                                })
                                setTimeout(function(){
                                    location.reload()
                                },1200)
                            }
                        }).catch(e=>{
                            console.error(e)
                            swal({
                                title:"请求出错,详细查看控制台",
                                icon:"error"
                            })
                        })
                    }
                })
            }
        }
    }

    Vue.createApp(vue_users).mount("#vue-users")
}



if(document.getElementById("vue-users-files")){
    const vue_users_files = {
        data(){
            return {

            }
        },
        methods:{
            download(url){
                swal({
                    title:"确认要下载吗?",
                    icon:"warning",
                    buttons:true
                }).then(r=>{
                    if(r){
                        download(url);
                    }
                })
            },
            remove(id){
                swal({
                    title:"确定要删除此文件吗?",
                    text:"删除后无法恢复",
                    icon:"warning",
                    buttons:true
                }).then(r=>{
                    if(r===true){
                        axios.post("/api/User/Files/remove",{
                            id:id,
                            _token:csrf_token
                        }).then(r => {
                            var data = r.data;
                            if(data.success===false){
                                swal({
                                    title:"Error",
                                    icon:"error",
                                    text:data.result.msg,
                                })
                            }else{
                                swal({
                                    title:"Success",
                                    icon:"success",
                                    text:data.result.msg,
                                })
                            }
                        }).catch(e=>{
                            console.error(e)
                            iziToast.error({
                                title:"Error",
                                message:"请求出错,详细查看控制台",
                                position:"topRight"
                            })
                        })
                    }
                })
            },
            alert(name){
                swal({
                    text:name
                })
            }
        }
    }
    Vue.createApp(vue_users_files).mount("#vue-users-files")
}


