import axios from "axios";
import iziToast from "izitoast";

// 评论点赞
$(function () {
    $('a[comment-click="comment-like-topic"]').click(function () {
        var comment_id = $(this).attr('comment-id');
        axios.post("/api/comment/like.topic.comment", {
            comment_id: comment_id,
            _token: csrf_token
        }).then(r => {
            const data = r.data;
            if (!data.success) {
                iziToast.error({
                    title: "error",
                    message: data.result.msg,
                    position: "topRight",
                    timeout: 10000
                })
            } else {
                var likes_text = $(this).children('span[comment-show="comment-topic-likes"]');
                var y_likes = likes_text.text();
                y_likes = parseInt(y_likes);
                if (data.code === 200) {
                    $(this).children('span[comment-show="comment-topic-likes"]').text(y_likes + 1)
                } else {
                    $(this).children('span[comment-show="comment-topic-likes"]').text(y_likes - 1)
                }

            }
        }).catch(e => {
            iziToast.error({
                title: "error",
                message: "请求出错,详细查看控制台",
                position: "topRight",
                timeout: 10000
            })
            console.error(e)
        })
    })
})

// 回复评论
$(function () {
    // 设置被回复评论ID
    $('a[comment-click="comment-reply-topic"]').click(function () {
        var comment_id = $(this).attr("comment-id");
        $("#reply-comment-id").val(comment_id)
    })
    // 发送回复评论请求
    $("#reply-comment-modal-reply-button").bind("click", function () {
        replyCommentRequest()
    })
    // ctrl enter 发布评论
    if (document.getElementById('reply-comment-content')) {
        let textarea = document.getElementById('reply-comment-content');
        // 添加键盘事件监听器
        textarea.addEventListener('keydown', function (event) {
            // 检查是否按下了Ctrl键并且按下了Enter键
            if ((event.ctrlKey && event.key === 'Enter') || (event.metaKey && event.key === 'Enter')) {
                // 阻止默认的Enter键行为（换行）
                event.preventDefault();
                replyCommentRequest()
            }
        });
    }

    function replyCommentRequest() {
        const comment_id = $("#reply-comment-id").val()
        if (!comment_id) {
            swal('Error', '回复评论的ID不存在，请刷新页面重试', 'error')
        }
        let rcc = $("#reply-comment-content");
        const content = rcc.val()
        if (!content) {
            swal('Error', '回复内容为空!', 'error')
        }
        rcc.attr("disabled", "disabled")
        axios.post("/api/comment/comment.topic.reply", {
            content: content,
            comment_id: comment_id,
            _token: csrf_token,
            originalContent: 1,
        }).then(r => {
            const data = r.data;
            if (data.success === true) {
                swal('Success', data.result.msg, 'success')
                $("#reply-comment-content").val(null)
                setTimeout(() => {
                    location.href = data.result.url
                }, 1200)
                return;
            }
            // 删除 rcc 的disabled 属性
            rcc.removeAttr("disabled")

            swal('Error', data.result.msg, 'error')
        }).catch(e => {
            // 删除 rcc 的disabled 属性
            rcc.removeAttr("disabled")
            console.error(e)
        })
    }
})

// 删除评论
$(function () {
    $('a[comment-click="comment-delete-topic"]').click(function () {
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
                    }).then(r => {
                        var data = r.data;
                        if (data.success === false) {
                            data.result.forEach(function (value) {
                                iziToast.error({
                                    title: "Error",
                                    message: value,
                                    position: "topRight"
                                })
                            })
                        } else {
                            data.result.forEach(function (value) {
                                iziToast.success({
                                    title: "Success",
                                    message: value,
                                    position: "topRight"
                                })
                            })
                            setTimeout(function () {
                                location.reload()
                            }, 1500)
                        }
                    }).catch(e => {
                        console.error(e)
                        iziToast.error({
                            title: "Error",
                            message: "请求出错,详细查看控制台",
                            position: "topRight"
                        })
                    })
                }
            });
    })
})

// 采纳评论
$(function () {
    $('a[comment-click="comment-caina-topic"]').click(function () {
        var th = $(this)
        var comment_id = th.attr("comment-id")
        axios.post("/api/comment/topic.caina.comment", {
            _token: csrf_token,
            comment_id: comment_id
        }).then(r => {
            var data = r.data;
            if (data.success === false) {
                data.result.forEach(function (value) {
                    iziToast.error({
                        title: "Error",
                        message: value,
                        position: "topRight"
                    })
                })
            } else {
                data.result.forEach(function (value) {
                    iziToast.success({
                        title: "Success",
                        message: value,
                        position: "topRight"
                    })
                })
                setTimeout(function () {
                    location.reload()
                }, 1500)
            }
        }).catch(e => {
            console.error(e)
            iziToast.error({
                title: "Error",
                message: "请求出错,详细查看控制台",
                position: "topRight"
            })
        })
    })
})

$(function () {
    $('a[comment-click="star-comment"]').click(function () {
        var th = $(this)
        var topic_id = th.attr("topic-id");
        var comment_id = th.attr("comment-id");
        axios.post("/api/comment/star.comment", {
            _token: csrf_token,
            topic_id: topic_id,
            comment_id: comment_id
        }).then(r => {
            if (!r.data.success) {
                iziToast.error({
                    title: "Error",
                    message: r.data.result.msg,
                    position: "topRight"
                })
            } else {
                iziToast.success({
                    title: "Success",
                    message: r.data.result.msg,
                    position: "topRight"
                })
            }
        }).catch(e => {
            console.error(e)
            iziToast.error({
                title: "Error",
                message: "请求出错,详细查看控制台",
                position: "topRight"
            })
        })
    })
})

// ctrl enter 发布评论
if (document.getElementById('create-comment-textarea')) {
    function filterHtml(str) {
        if (typeof str != 'string') {  //不是字符串
            return str;
        }

        return str.replace(reg, '');
    }

    let textarea = document.getElementById('create-comment-textarea');
    // 设置textarea的value
    textarea.value = filterHtml(textarea.value);

    // 添加键盘事件监听器
    textarea.addEventListener('keydown', function (event) {
        // 检查是否按下了Ctrl键并且按下了Enter键
        if ((event.ctrlKey && event.key === 'Enter') || (event.metaKey && event.key === 'Enter')) {
            // 阻止默认的Enter键行为（换行）
            event.preventDefault();
            // 创建一个隐藏的 input 元素
            let hiddenInput = document.createElement("input");
            hiddenInput.type = "hidden";
            hiddenInput.name = "content";
            hiddenInput.value = textarea.value;

            // 在 textarea 元素之前插入隐藏的 input 元素
            textarea.parentNode.insertBefore(hiddenInput, textarea);
            // 设置 textarea为disabled
            textarea.setAttribute('disabled', 'disabled');
            // 提交表单
            let form = document.getElementById('topic-create-comment-form');
            form.submit();
        }
    });

}