import Vditor from "vditor";
import axios from "axios";
import iziToast from "izitoast";
import tippy, {animateFill} from 'tippy.js';

if(document.getElementById("topic-comment-model")){
    const topic_comment = {
        data(){
            return {
                topic_id:topic_id,
                vditor: '',
                edit:{
                    mode:"ir",
                    preview:{
                        mode:"editor"
                    }
                },
                tag_selected:1,
                userAtList:[

                ],
                topic_keywords:[

                ],
            }
        },

        methods:{

            selectEmoji(emoji){
                emoji = " "+ emoji + " "
                this.vditor.insertValue(emoji)
                this.vditor.tip("表情插入成功!")
            },
            submit(){
                const content = this.vditor.getHTML();
                const markdown = this.vditor.getValue()
                if(!content || !markdown){
                    iziToast.error({
                        title:"Error",
                        message:"评论内容不能为空!",
                        position: 'topRight',
                    })
                    return ;
                }
                axios.post("/api/comment/topic.create",{
                    content:content,
                    markdown:markdown,
                    _token:csrf_token,
                    topic_id:this.topic_id
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
                        this.vditor.clearCache()
                        iziToast.success({
                            title:"Success",
                            message:data.result.msg,
                            position:"topRight",
                            timeout:1000
                        })
                        setTimeout(()=>{
                            location.href=data.result.url;
                        },1000);
                    }
                }).catch(e=>{
                    iziToast.error({
                        title:"Error",
                        message:"请求出错,详细查看控制台",
                        position:"topRight"
                    })
                    console.error(e)
                })
            },

        },

        mounted () {

            // vditor
            this.vditor = new Vditor('topic-comment', {
                cdn:'/js/vditor',
                height: 300,
                toolbarConfig: {
                    pin: true,
                },
                cache: {
                    enable: true,
                    id: "create_comment_"+this.topic_id
                },
                preview: {
                    markdown: {
                        toc: false,
                        mark: true,
                        autoSpace: true
                    }
                },
                mode: this.edit.mode,
                toolbar: [
                    "emoji",
                    "headings",
                    "bold",
                    "italic",
                    "strike",
                    "link",
                    "|",
                    "quote",
                    "line",
                    "code",
                    "inline-code",
                    "insert-before",
                    "insert-after",
                    "|",
                    "table",
                    "undo",
                    "redo",
                    "|",
                    "edit-mode",

                ],
                counter: {
                    "enable": true,
                    "type": "已写字数"
                },
                hint: {
                    extend: [
                        {
                            key: '@',
                            hint: (key) => {
                                return this.userAtList
                            },
                        }, ],
                },

                typewriterMode: true,
                placeholder: "请输入评论内容",
                after: () => {
                    axios.post("/api/user/@user_list",{_token:csrf_token})
                        .then(r=>{
                            this.userAtList =  r.data;
                        })
                        .catch(e=>{
                            swal({
                                title:"获取本站用户列表失败,详细查看控制台",
                                icon:"error"
                            })
                            console.error(e)
                        })
                    axios.post("/api/topic/keywords",{_token:csrf_token})
                        .then(r=>{
                            this.topic_keywords =  r.data;
                        })
                        .catch(e=>{
                            swal({
                                title:"获取话题列表失败,详细查看控制台",
                                icon:"error"
                            })
                            console.error(e)
                        })
                },
                input(md){

                },
                select:(md) => {

                }
            });


        },
    }
    Vue.createApp(topic_comment).mount("#topic-comment-model")
}

$(function(){
    $('div[comment-load="topic"]').each(function(){
        var topic_id = $(this).attr("topic-id")
        var comment = $(this);
        axios.post("/api/comment/get.topic.comment",{
            _token:csrf_token,
            topic_id:topic_id,
        }).then(r=>{
            var data = r.data
            console.log(data)
            $(this).children('.row').empty();
            if(data.success===false){
                return ;
            }
            data = data.result
            data.data.forEach(function(value,index){
                comment.children('.row.row-cards').append(comment_singer(value,index))
            })
        })
    })

    function comment_singer(data,index){
        var lc = index+1;
        var user_avatar = null;
        $.post("/api/user/get.user.avatar.url",{
            _token:csrf_token,
            user_id : data.user_id
        },function(data){
            user_avatar = data.msg
        })
        return `
<div class="col-md-12" id="#comment-${data.id}">
    <div class="border-0 card card-body">
        <div class="row">
            <div class="col-md-12">
                <div class="row">
                   
                    <div class="col text-truncate">
                        <a href="" class="text-body d-block text-truncate">${data.user.username}</a>
                        <small class="text-muted text-truncate mt-n1">发表于:${data.created_at}</small>
                    </div>
                    <div class="col-auto">
                        <a href="/${data.topic_id}.html#comment-${data.id}">${lc}楼</a>
                    </div>
                </div>
            </div>
            <div class="col-md-12">
                <div class="hr-text" style="margin-bottom:5px;margin-top:15px">评论内容</div>
            </div>
            <div class="col-md-12 markdown vditor-reset overflow-auto">
                ${data.content}
            </div>
            <div class="col-md-12">
                <div class="hr-text" style="margin-bottom:5px;margin-top:15px">操作</div>
            </div>
            <div class="col-md-12">
            <a data-bs-toggle="tooltip" data-bs-placement="top" title="" href="/comment/topic/${data.id}.md" class="switch-icon switch-icon-flip" data-bs-original-title="查看markdown文本">
                            <span class="switch-icon-a text-muted">
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-markdown" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                    <path stroke="none" d="M0 0h24v24H0z" fill="none">
                                    </path>
                                    <rect x="3" y="5" width="18" height="14" rx="2">
                                    </rect>
                                    <path d="M7 15v-6l2 2l2 -2v6"></path>
                                    <path d="M14 13l2 2l2 -2m-2 2v-6"></path>
                                </svg>
                            </span>
                        </a>
            </div>
        </div>
    </div>
</div>
        `;
    }
})
const template = document.getElementById('template');

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
        var th = $(this)
        var comment_id = th.attr("comment-id");
        var dom = $('div[comment-dom="comment-'+comment_id+'"]')
        var comment_status = dom.attr("comment-status");
        var vditor_dom = $('div[comment-dom="comment-vditor-'+comment_id+'"]')
        var comment_url = vditor_dom.attr("comment-url")
        if(vditor_dom.attr("comment-status")==="off"){
            var vditor = new Vditor('comment-reply-vditor-'+comment_id, {
                cdn:'/js/vditor',
                height: 250,
                toolbarConfig: {
                    pin: true,
                },
                typewriterMode: true,
                placeholder: "请输入回复内容",
                cache: {
                    enable: false,
                },
                toolbar: [
                    "emoji",
                    "headings",
                    "bold",
                    "italic",
                    "strike",
                    "link",
                    "|",
                    "quote",
                    "line",
                    "code",
                    "inline-code",
                    "insert-before",
                    "insert-after",
                    "|",
                    "table",
                    "undo",
                    "redo",
                    "|",
                    "edit-mode",

                ],
                counter: {
                    "enable": true,
                    "type": "已写字数"
                },
                after: () => {

                },
            })
            vditor_dom.attr("comment-status","on")
        }

        if(comment_status==="off"){
            dom.attr("comment-status","on")
            dom.children(".hr-text").show();
            vditor_dom.show()
            $('button[comment-dom="comment-vditor-submit-'+comment_id+'"]').click(function(){
                var content =vditor.getHTML();
                var markdown =vditor.getValue();
                if(!content || !markdown){
                    iziToast.error({
                        title:"Error",
                        message:"回复内容不能为空!",
                        position:"topRight"
                    })
                    return ;
                }
                axios.post("/api/comment/comment.topic.reply", {
                    _token: csrf_token,
                    comment_id: comment_id,
                    content: content,
                    markdown: markdown,
                    parent_url:comment_url
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
                        iziToast.success({
                            title:"Success",
                            message:data.result.msg,
                            position:"topRight",
                            timeout:1000
                        })
                        setTimeout(()=>{
                            location.href=data.result.url;
                        },1000);
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
            return;
        }
        dom.attr("comment-status","off")
        dom.children(".hr-text").hide();
        vditor_dom.hide()


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