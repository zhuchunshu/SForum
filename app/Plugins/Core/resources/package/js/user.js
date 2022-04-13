import axios from "axios";
import iziToast from "izitoast";

if(document.getElementById("vue-user-my-setting")){
    const vums = {
        data(){
            return {
                username : "",
                email:"",
                old_pwd:"",
                new_pwd:"",
                avatar:"",
                avatar_url :"",
            }
        },
        methods:{
            getFile(event) {
                this.avatar = event.target.files[0];
            },
            submit(event){
                event.preventDefault();
                axios.post("/user/myUpdate",{
                    _token: csrf_token,
                    username:this.username,
                    old_pwd:this.old_pwd,
                    new_pwd:this.new_pwd,
                    avatar:this.avatar,
                }).then(r => {
                    const data = r.data;
                    console.log(data);
                })
            }
        },
        beforeMount() {
            axios.post("/user/data",{_token:csrf_token})
                .then(response => {
                    this.username = response.data.username;
                    this.email = response.data.email;

                })
                .catch(error => {
                    swal({
                        title:"个人信息获取失败,详细查看控制台",
                        icon:"error"
                    })
                    console.error(error)
                })
        }
    }
    Vue.createApp(vums).mount("#vue-user-my-setting")
}


// 用户通知
$(function(){
    // action
    $('a[user-click="notice_action"]').click(function(){
        var href = $(this).attr("notice-href")
        var notice_id = $(this).attr("notice-id")
        axios.post("/api/user/notice.read",{
            _token:csrf_token,
            notice_id:notice_id,
        }).then(r=>{
            location.href=href
        })
    })
    // 确认已读
    $('button[user-click="notice_read"]').click(function(){
        var notice_id = $(this).attr("notice-id")
        axios.post("/api/user/notice.read",{
            _token:csrf_token,
            notice_id:notice_id,
        }).then(r=>{
            var data = r.data
            if(data.success===false){
                iziToast.error({
                    title: "Error",
                    position:"topRight",
                    message:data.result.msg
                })
            }else{
                iziToast.success({
                    title: "Success",
                    position:"topRight",
                    message:data.result.msg
                })
                setTimeout(function(){
                    location.reload()
                },1500)
            }
        })
    })
})

$(function () {
    // 关注用户
    $('a[user-click="user_follow"]').click(function(){
        var user_id = $(this).attr("user-id")
        var th = $(this)
        axios.post("/api/user/userfollow",{
            _token:csrf_token,
            user_id:user_id
        }).then(r=>{
            var data = r.data;
            if(data.success=== true){
                if(data.code===200){
                    th.children('span').text(data.result.msg)
                }else{
                    th.children('span').text("关注")
                }
                iziToast.success({
                    title:"Success",
                    message:data.result.msg,
                    position:"topRight"
                })
            }else{
                iziToast.error({
                    title:"Error",
                    message:data.result.msg,
                    position:"topRight"
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
    })

    //查询关注状态
    $('a[user-click="user_follow"]').each(function(){
        var user_id = $(this).attr("user-id")
        var th = $(this)
        axios.post("/api/user/userfollow.data",{
            _token:csrf_token,
            user_id:user_id
        }).then(r=>{
            var data = r.data;
            if(data.success=== true){
                th.children('span').text(data.result.msg)
            }
        }).catch(e=>{
            console.error(e)
            iziToast.error({
                title:"Error",
                message:"请求出错,详细查看控制台",
                position:"topRight"
            })
        })
    })
})

$(function(){
    $("button[user-click=\"remove_collections\"]").click(function(){
        var collection_id = $(this).attr("collections-id");
        axios.post("/api/user/remove.collection",{
            _token:csrf_token,
            collection_id:collection_id
        }).then(r=>{
            if(!r.data.success){
                iziToast.error({
                    title:"Error",
                    message:r.data.result.msg,
                    position:"topRight"
                })
            }else{
                iziToast.success({
                    title:"Success",
                    message:r.data.result.msg,
                    position:"topRight"
                })
                setTimeout(function(){
                    location.reload();
                },1500)
            }
        }).catch(e=>{
            console.error(e)
            iziToast.error({
                title:"Error",
                message:"请求出错,详细查看控制台",
                position:"topRight"
            })
        })
    })
})