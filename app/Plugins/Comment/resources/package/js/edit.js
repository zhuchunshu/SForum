import Vditor from "vditor";
import axios from "axios";
import iziToast from "izitoast";

if(document.getElementById("vue-comment-topic-edit-form")){
    const vue = {
        data() {
            return {
                vditor: '',
                comment_id:comment_id,
                topic_id:topic_id
            }
        },
        mounted() {
            // vditor
            this.vditor = new Vditor('vditor', {
                height: 300,
                toolbarConfig: {
                    pin: true,
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
                    axios.post("/api/comment/topic.comment.data",{
                        _token:csrf_token,
                        comment_id:this.comment_id
                    }).then(r => {
                        const data = r.data
                        if(data.success===false){
                            data.result.forEach(function(value){
                                iziToast.error({
                                    title:"Error",
                                    message:value
                                })
                            })
                        }else{
                            this.vditor.setValue(data.result.markdown)
                        }
                    }).catch(e=>{
                        iziToast.error({
                            title:"Error",
                            message:"请求出错,详细查看控制台",
                            position:"topRight"
                        })
                        console.error(e)
                    })
                }
            });
        },
        methods: {
            submit() {
                var content = this.vditor.getHTML();
                var markdown = this.vditor.getValue();
                axios.post("/api/comment/topic.comment.update",{
                    _token:csrf_token,
                    content:content,
                    markdown:markdown,
                    comment_id:this.comment_id
                }).then(r=>{
                    const data = r.data;
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
                            location.href="/"+this.topic_id+".html"
                        },1500)
                    }
                }).catch(e=>{
                    iziToast.error({
                        title:"Error",
                        message:"请求出错,详细查看控制台",
                        position:"topRight"
                    })
                    console.error(e)
                })
            }
        }
    }
    Vue.createApp(vue).mount("#vue-comment-topic-edit-form")
}