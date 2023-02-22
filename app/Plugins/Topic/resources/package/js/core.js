import axios from "axios";
import iziToast from "izitoast";
import swal from "sweetalert";
import copy from 'copy-to-clipboard';

$(function(){
    $('div[core-data="topic"]').each(function(){
        const y = $(this);
        const content = $(this).find('div[core-data="topic_content"]');
        const author =  $(this).find('div[core-data="topic-author"]');
        const topic_id = y.attr("topic-id");
        axios.post("/api/topic/with_topic.data",{
            topic_id: topic_id,
            _token:csrf_token
        }).then(r=>{
            const data = r.data;
            if(data.success===false){
                data.result.forEach(function(value){
                    iziToast.error({
                        title:"error",
                        message:value,
                        position:"topRight",
                        timeout:10000
                    })
                })
            }else{
                const topic = data.result;

                //设置帖子内容
                content.html(`
                <a href="/${topic_id}.html" class="text-reset" style="text-decoration:none;"><b>${topic.title}</b></a>
        <a href="/${topic_id}.html" style="display: -webkit-box;
    font-size: 13px;
    height: 18px;
    line-height: 18px;
    color: #999999;
    word-break: break-all;
    text-overflow: ellipsis;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 1;
    text-decoration:none;">
            ${topic.summary}
        </a>
                `)
                // 设置用户头像
                author.html(`
                <a href="/users/${topic.username}.username" class="avatar" style="background-image: url(${topic.avatar})"></a>
                `)
            }
        }).catch(e=>{
            y.html("ID为:"+topic_id+"获取失败!,详细查看控制台");
            console.error(e)
        })
    });
})

if(document.getElementById('topic-page')){
    $(function(){
        $("#topic-page img").each(function(){
            const img = $(this);
            if(img.attr("lightbox")!=="false" && img.parents().get(0).tagName!=="a" && img.parents().get(0).tagName!=="A"){
                const img_url = img.attr("src");
                const img_alt = img.attr("alt");
                img.wrap("<a data-fslightbox=\"gallery\" href='"+img_url+"'></a>")
            }
        })

        $("#topic-page a").each(function(){
            const a = $(this);
            if(a.children().length>0){
                if(a.children().get(0).tagName==="IMG" || a.children().get(0).tagName==="img"){
                    a.attr("data-fslightbox","gallery")
                    const img_alt = a.children().attr("alt")
                    a.attr("data-caption",img_alt)
                }
            }

        })

        refreshFsLightbox();
    })

}
$(function(){
    $('a[core-click="like-topic"]').click(function(){
        const topic_id = $(this).attr("topic-id")
        axios.post("/api/topic/like.topic", {
            topic_id: topic_id,
            _token: csrf_token
        }).then(r  =>{
            const data = r.data;
            if(!data.success){
                iziToast.error({
                    title:"error",
                    message:data.result.msg,
                    position:"topRight",
                    timeout:10000
                })
            }else{
                // 点赞成功!
                var likes_text = $(this).children('span[core-show="topic-likes"]');
                var y_likes = likes_text.text();
                y_likes = parseInt(y_likes);
                if(data.code===200){
                    $(this).children('span[core-show="topic-likes"]').text(y_likes+1)
                }else{
                    $(this).children('span[core-show="topic-likes"]').text(y_likes-1)
                }
            }
        }).catch(e=>{
            iziToast.error({
                title:"error",
                message:"请求出错,详细查看控制台",
                position:"topRight",
                timeout:10000
            })
            console.error(e)
        })
    })
})

$(function(){
    $('span[core-click="user-avatar"]').click(function(){
        location.href="/users/"+$(this).attr("username")+".username"
    })
})

$(function(){
    if(!window.location.hash && typeof comment_id!=="undefined"){
        window.location.hash="#comment-"+comment_id
        var target = $(location.hash);
        if(target.length===1){
            var top = target.offset().top-200;
            $('html,body').animate({scrollTop:top+"px"}, 1000);
        }
        var url = document.URL
        url = url.replace(window.location.hash,"")
        history.replaceState(null,document.title,url)
    }
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
    // 收藏帖子
    $('a[core-click="star-topic"]').click(function(){
        var th = $(this);
        var topic_id = th.attr("topic-id");
        axios.post("/api/topic/star.topic",{
            _token:csrf_token,
            topic_id: topic_id,
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




// 举报
$(function(){

    // 举报帖子
    $('a[core-click="report-topic"]').click(function(){
        $("#modal-report-title").text("举报此帖")
        $("#modal-report-input-type").val("topic")
        $("#modal-report-input-type-id").val($(this).attr("topic-id"))
        $("#modal-report-input-content").val("违规页面地址:"+location.href)
        $("#modal-report-input-url").val(location.href)
        var selected=$("#modal-report-select").val();
        $("#modal-report-input-title").val("【"+selected+"】"+"举报ID为:"+$(this).attr("topic-id")+"的帖子")
    })

    // 举报评论
    $('a[comment-click="report-comment"]').click(function(){
        $("#modal-report-title").text("举报此评论")
        $("#modal-report-input-type").val("comment")
        $("#modal-report-input-type-id").val($(this).attr("comment-id"))
        $("#modal-report-input-content").val("违规页面地址:"+location.protocol+"//"+location.host+$(this).attr("url"))
        var selected=$("#modal-report-select").val();
        $("#modal-report-input-url").val(location.protocol+"//"+location.host+$(this).attr("url"))
        $("#modal-report-input-title").val("【"+selected+"】"+"举报ID为:"+$(this).attr("comment-id")+"的评论")
    })

    // 引用评论
    $('a[comment-click="quote-comment"]').click(function(){
        const comment_id = $(this).attr("comment-id");
        const shortCode="[topic-comment="+comment_id+"][/topic-comment]";
        copy(shortCode);
        iziToast.success({
            title:"Success",
            message:"已复制短代码!",
            position:"topRight"
        })
    })

    // select 变化事件
    $("#modal-report-select").change(function(){
        var selected=$(this).children('option:selected').val();
        $("#modal-report-input-title").val("【"+selected+"】"+removeBlock($("#modal-report-input-title").val()))
    })


    //关闭modal
    $("button[modal-click=\"close\"]").click(function () {
        $("#modal-report-title").text(null)
        $("#modal-report-input-type").val(null)
        $("#modal-report-input-type-id").val(null)
        $("#modal-report-input-content").val(null)
        $("#modal-report-input-title").val(null)
        $("#modal-report-input-url").val(null);
    })

    // 提交举报
    $('#modal-report-submit').click(function(){
        // 举报类型
        var type = $("#modal-report-input-type").val();
        var type_id = $("#modal-report-input-type-id").val();
        // 举报标题
        var title = $("#modal-report-input-title").val();
        // 举报详细内容
        var content = $("#modal-report-input-content").val();
        // 举报原因
        var report_reason = $("#modal-report-select").val();
        // 相关链接
        var url = $("#modal-report-input-url").val();
        axios.post("/api/core/report/create",{
            _token:csrf_token,
            type:type,
            type_id:type_id,
            title:title,
            content:content,
            report_reason:report_reason,
            url:url
        }).then(r=>{
            if(!r.data.success){
                r.data.result.forEach(function(value){
                    iziToast.error({
                        title:"error",
                        message:value,
                        position:"topRight",
                        timeout:10000
                    })
                })
            }else{
                r.data.result.forEach(function(value){
                    iziToast.success({
                        title:"Success",
                        message:value,
                        position:"topRight"
                    })
                })
            }
        }).catch(e=>{
            console.error(e)
            iziToast.error({
                position:"topRight",
                title:"Error",
                message:"请求出错,详细查看控制台"
            })
        })
    })
})

function removeBlock(str) {
    if (str) {
        var reg = /\【.*\】/gi;
        str = str.replace(reg, "");
    }
    return str;
}


// 加载帖子更新记录作者IP归属地

$(function(){
    let updateds = [];
    $('span[topic-type="updated_ip"]').each(function(){
        updateds.push($(this).attr("updated-id"));
    })
    if(updateds.length>0){
        axios.post('/api/topic/get.updated.user.ip',{
            _token:csrf_token,
            updateds:updateds
        }).then(r=>{
            let data = r.data;
            data = data.result
            data.forEach(function(v){
                $('span[updated-id="'+v.updated_id+'"]').text(v.text);
            })
        })
    }
})


// 对帖子页面的操作
$(function(){

    // 精华
    $('a[core-click="topic-essence"]').click(function(){
        var topic_id = $(this).attr("topic-id");
        swal({
            title:"精华指数,数字越大排名越靠前,0为取消加精",
            content: {
                element: "input",
                attributes: {
                    type: "number",
                    max:999,
                    min:1
                },
            },
        }).then(r => {
            if(r && !isNaN(r) && r>=0){
                axios.post("/api/topic/set.topic.essence",{
                    _token:csrf_token,
                    topic_id:topic_id,
                    zhishu:r
                }).then(r=>{
                    const data = r.data;
                    if(data.success){
                        iziToast.success({
                            title: 'Success',
                            position: 'topRight',
                            message: data.result.msg,
                        });
                    }else{
                        iziToast.error({
                            title: 'Error',
                            position: 'topRight',
                            message: data.result.msg,
                        });
                    }
                }).catch(e=>{
                    iziToast.error({
                        title: 'Error',
                        position: 'topRight',
                        message: '请求出错,详细查看控制台',
                    });
                    console.error(e)
                })
            }
        });
    })

    // 置顶
    $('a[core-click="topic-topping"]').click(function(){
        var topic_id = $(this).attr("topic-id");
        swal({
            title:"置顶指数,数字越大排名越靠前,0为取消置顶",
            content: {
                element: "input",
                attributes: {
                    type: "number",
                    max:999,
                    min:1
                },
            },
        }).then(r => {
            if(r && !isNaN(r) && r>=0){
                axios.post("/api/topic/set.topic.topping",{
                    _token:csrf_token,
                    topic_id:topic_id,
                    zhishu:r
                }).then(r=>{
                    const data = r.data;
                    if(data.success){
                        iziToast.success({
                            title: 'Success',
                            position: 'topRight',
                            message: data.result.msg,
                        });
                    }else{
                        iziToast.error({
                            title: 'Error',
                            position: 'topRight',
                            message: data.result.msg,
                        });
                    }
                }).catch(e=>{
                    iziToast.error({
                        title: 'Error',
                        position: 'topRight',
                        message: '请求出错,详细查看控制台',
                    });
                    console.error(e)
                })
            }
        });
    })


    // 锁帖
    $('a[core-click="topic-lock"]').click(function(){
        var topic_id = $(this).attr("topic-id");
        axios.post("/api/topic/set.topic.lock",{
            _token:csrf_token,
            topic_id:topic_id,
        }).then(r=>{
            const data = r.data;
            if(data.success){
                iziToast.success({
                    title: 'Success',
                    position: 'topRight',
                    message: data.result.msg,
                });
            }else{
                iziToast.error({
                    title: 'Error',
                    position: 'topRight',
                    message: data.result.msg,
                });
            }
        }).catch(e=>{
            iziToast.error({
                title: 'Error',
                position: 'topRight',
                message: '请求出错,详细查看控制台',
            });
            console.error(e)
        })
    })

    // 删除
    $('a[core-click="topic-delete"]').click(function(){
        var topic_id = $(this).attr("topic-id");
        swal({
            title:"确定要删除此贴吗? 删除后不可恢复",
            buttons: ["取消", "确定"],
        }).then(r => {
            if(r===true){
                axios.post("/api/topic/set.topic.delete",{
                    _token:csrf_token,
                    topic_id:topic_id,
                    zhishu:r
                }).then(r=>{
                    const data = r.data;
                    if(data.success){
                        iziToast.success({
                            title: 'Success',
                            position: 'topRight',
                            message: data.result.msg,
                        });
                    }else{
                        iziToast.error({
                            title: 'Error',
                            position: 'topRight',
                            message: data.result.msg,
                        });
                    }
                }).catch(e=>{
                    iziToast.error({
                        title: 'Error',
                        position: 'topRight',
                        message: '请求出错,详细查看控制台',
                    });
                    console.error(e)
                })
            }
        });
    })
})

if(document.getElementById("author")){
    const author = {
        data(){
            return {
                'user':{
                    'city':null,
                }
            }
        },
        mounted(){
            this.getUserCity();
        },
        methods:{
            // 获取作者所在城市
            getUserCity(){
                axios.post("/api/topic/get.user",{
                    _token:csrf_token,
                    topic_id:topic_id
                }).then(r=>{
                    this.user = r.data.result;
                }).catch(e=>{
                    iziToast.error({
                        title: 'Error',
                        message:"请求出错,详细查看控制台",
                        position:"topRight"
                    })
                    console.error(e)
                })
            }
        }


    }

    Vue.createApp(author).mount('#author');
}


// 加载评论作者IP归属地
$(function(){
    let comments = [];
    $('small[comment-type="ip"]').each(function(){
        comments.push($(this).attr("comment-id"));
    })
    if(comments.length>0){
        axios.post('/api/comment/get.user.ip',{
            _token:csrf_token,
            comments:comments
        }).then(r=>{
            let data = r.data;
            data = data.result
            data.forEach(function(v){
                $('small[comment-id="'+v.comment_id+'"]').text(v.text);
            })
        })
    }
})
