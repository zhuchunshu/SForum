import axios from "axios";
import swal from "sweetalert";
import iziToast from "izitoast";


if(document.getElementById("vue-header-right-my")){
    const vhrm ={
        data(){
            return {

            }
        },
        methods:{
            Logout(){
                axios
                    .post("/logout",{_token:csrf_token})
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
$(function(){
    $("a[menu=active]").each(function(){
        const d = $(this);
        const _class =d.parent().parent().parent().parent().attr("class");
        d.parent().parent().parent().parent().attr('class', _class + " active")
    })
})

$(function(){
    $("img").each(function(){
        if($(this).attr("data-src")){
            var d = $(this)
            setTimeout(function(){
                d.attr("src",d.attr("data-src"))
            },500)
        }
    })
})

console.log("%cSuperForum %cwww.github.com/zhuchunshu/super-forum","color:#fff;background:linear-gradient(90deg,#448bff,#44e9ff);padding:5px 0;", "color:#000;background:linear-gradient(90deg,#44e9ff,#ffffff);padding:5px 10px 5px 0px;")

$(function(){
    axios.post("/api/user/get.user.config",{
        _token:csrf_token
    }).then(r=>{
        var data = r.data;
        if(data.success===false){
            return ;
        }
        data = data.result;

        // 通知小红点
        if (document.getElementById("core-notice-red")){
            if(data.notice_red===true){
                $("#core-notice-red").show();
            }
        }
    })
})

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
$(function(){
    $("#core_update_theme").click(function(){
        let theme = $("body").attr("class");
        if(theme==="antialiased"){
            $("body").attr("class",'theme-dark')
            $(this).html('<svg class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M0 0h24v24H0z" stroke="none"></path><circle cx="12" cy="12" r="4"></circle><path d="M3 12h1m8-9v1m8 8h1m-9 8v1M5.6 5.6l.7.7m12.1-.7l-.7.7m0 11.4l.7.7m-12.1-.7l-.7.7"></path></svg>');

        }
        if(theme==="theme-dark"){
            $("body").attr("class",'antialiased')
            $(this).html('<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-moon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">\n' +
                '                    <desc>Download more icon variants from https://tabler-icons.io/i/moon</desc>\n' +
                '                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>\n' +
                '                    <path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z"></path>\n' +
                '                </svg>');
        }
        axios.post("/api/core/toggle.theme",{
            _token:csrf_token,
            theme:$("body").attr("class")
        }).then(r=>{
            let data = r.data;
            if(data.success===false){
                iziToast.error({
                    title: "Error",
                    message:data.result.msg,
                    position:"topRight"
                })
            }else{
                iziToast.success({
                    title: "Success",
                    message:data.result.msg,
                    position:"topRight"
                })
            }
        })
    })
})