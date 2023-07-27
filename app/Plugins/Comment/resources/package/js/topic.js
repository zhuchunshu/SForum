import axios from "axios";
import iziToast from "izitoast";

// 评论点赞
$(function(){
    $('a[comment-click="comment-like-topic"]').click(function(){
        var comment_id = $(this).attr('comment-id');
        axios.post("/api/comment/like.topic.comment", {
            comment_id: comment_id,
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
                var likes_text = $(this).children('span[comment-show="comment-topic-likes"]');
                var y_likes = likes_text.text();
                y_likes = parseInt(y_likes);
                if(data.code===200){
                    $(this).children('span[comment-show="comment-topic-likes"]').text(y_likes+1)
                }else{
                    $(this).children('span[comment-show="comment-topic-likes"]').text(y_likes-1)
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

// 回复评论
$(function(){
    $('a[comment-click="comment-reply-topic"]').click(function(){
        var comment_id = $(this).attr("comment-id");
        $("#reply-comment-id").val(comment_id)
    })
})
$(function(){
    $("#reply-comment-modal-reply-button").bind("click",function(){
        const comment_id = $("#reply-comment-id").val()
        if(!comment_id){
            swal('Error','回复评论的ID不存在，请刷新页面重试','error')
        }
        const content = $("#reply-comment-content").val()
        if(!content){
            swal('Error','回复内容为空!','error')
        }
        axios.post("/api/comment/comment.topic.reply",{
            content:content,
            comment_id:comment_id,
            _token:csrf_token,
            originalContent:1,
        }).then(r=>{
            const data = r.data;
            if(data.success===true){
                swal('Success',data.result.msg,'success')
                $("#reply-comment-content").val(null)
                setTimeout(()=>{
                    location.href=data.result.url
                },1200)
                return ;
            }
            swal('Error',data.result.msg,'error')
        }).catch(e=>{
            console.error(e)
        })
    })
})

// 删除评论
$(function(){
    $('a[comment-click="comment-delete-topic"]').click(function() {
        var th = $(this)
        var comment_id = th.attr("comment-id")
        console.log(comment_id)
        swal({
            title: "确定要删除此评论吗?",
            text: "删除后不可恢复,请谨慎操作",
            icon: "warning",
            buttons: true,
            dangerMode: true,
        })
            .then((willDelete) => {
                if (willDelete) {
                    axios.post("/api/comment/comment.topic.delete", {
                        _token: csrf_token,
                        comment_id: comment_id,
                    }).then (r =>{
                        var data = r.data;
                        if(data.success===false){
                            data.result.forEach(function(value){
                                iziToast.error({
                                    title:"Error",
                                    message:value,
                                    position:"topRight"
                                })
                            })
                        }else{
                            data.result.forEach(function(value){
                                iziToast.success({
                                    title:"Success",
                                    message:value,
                                    position:"topRight"
                                })
                            })
                            setTimeout(function(){
                                location.reload()
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
                }
            });
    })
})

// 采纳评论
$(function(){
    $('a[comment-click="comment-caina-topic"]').click(function(){
        var th = $(this)
        var comment_id = th.attr("comment-id")
        axios.post("/api/comment/topic.caina.comment",{
            _token:csrf_token,
            comment_id:comment_id
        }).then(r=>{
            var data = r.data;
            if(data.success===false){
                data.result.forEach(function(value){
                    iziToast.error({
                        title:"Error",
                        message:value,
                        position:"topRight"
                    })
                })
            }else{
                data.result.forEach(function(value){
                    iziToast.success({
                        title:"Success",
                        message:value,
                        position:"topRight"
                    })
                })
                setTimeout(function(){
                    location.reload()
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

$(function(){
    $('a[comment-click="star-comment"]').click(function(){
        var th = $(this)
        var topic_id = th.attr("topic-id");
        var comment_id = th.attr("comment-id");
        axios.post("/api/comment/star.comment",{
            _token:csrf_token,
            topic_id: topic_id,
            comment_id:comment_id
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