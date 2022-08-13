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
                cdn:'/js/vditor',
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
                    {
                        hotkey: '⌘-⇧-V',
                        name:'video',
                        tipPosition: 's',
                        tip: '插入视频',
                        className: 'right',
                        icon: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-camera-video" viewBox="0 0 16 16">\n' +
                            '  <path fill-rule="evenodd" d="M0 5a2 2 0 0 1 2-2h7.5a2 2 0 0 1 1.983 1.738l3.11-1.382A1 1 0 0 1 16 4.269v7.462a1 1 0 0 1-1.406.913l-3.111-1.382A2 2 0 0 1 9.5 13H2a2 2 0 0 1-2-2V5zm11.5 5.175 3.5 1.556V4.269l-3.5 1.556v4.35zM2 4a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h7.5a1 1 0 0 0 1-1V5a1 1 0 0 0-1-1H2z"/>\n' +
                            '</svg>',
                        click :() => {
                            swal("请输入视频链接:", {
                                content: "input",
                            })
                                .then((value) => {
                                    if(value){
                                        let content = '<video  controls>\n' +
                                            '  <source src="'+value+'" ">\n' +
                                            '</video>'
                                        this.vditor.focus();
                                        this.vditor.insertValue(content);
                                    }
                                });
                        }
                    },
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
            selectEmoji(emoji){
                emoji = " "+ emoji + " "
                this.vditor.insertValue(emoji)
                this.vditor.tip("表情插入成功!")
            },

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