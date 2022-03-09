import Vditor from "vditor";
import axios from "axios";
import iziToast from "izitoast";

if(document.getElementById('vue-blog-article-create')){
    const app = {
        data(){
            return {
                vditor:null,
                title:localStorage.getItem("blog_article_create_title"),
                class_id:'none',
                classList:null
            }
        },
        mounted(){
            if(localStorage.getItem("blog_article_create_class_id")){
                this.class_id = localStorage.getItem("blog_article_create_class_id");
            }
            this.init()
        },
        methods:{
            init(){
                axios.post("/api/Blog/article/class",{_token:csrf_token})
                    .then(response => {
                        this.classList = response.data
                    })
                    .catch(e=>{
                        console.error(e)
                    })
                // vditor
                this.vditor = new Vditor('content-vditor', {
                    height: 400,
                    toolbarConfig: {
                        pin: true,
                    },
                    cache: {
                        enable: true,
                        id: "create_myblog_article"
                    },
                    preview: {
                        markdown: {
                            toc: true,
                            mark: true,
                            autoSpace: true
                        }
                    },
                    toolbar: [
                        "emoji",
                        "headings",
                        "bold",
                        "italic",
                        "strike",
                        "link",
                        "|",
                        "list",
                        "ordered-list",
                        "|",
                        "line",
                        "insert-before",
                        "insert-after",
                        "table",
                        "|",
                        "fullscreen",
                        "edit-mode",

                    ],
                    counter: {
                        "enable": true,
                        "type": "已写字数"
                    },

                    typewriterMode: true,
                    placeholder: "请输入正文",
                    input(md){

                    },
                    select:(md) => {

                    }
                });
            },
            submit(){
                const title = this.title;
                const class_id = this.class_id;
                const content = this.vditor.getHTML();
                const markdown = this.vditor.getValue();
                if(!isNumber(class_id)){
                    swal({
                        title:"请选择有效的分类",
                        icon:"error"
                    })
                    return ;
                }
                if(!title){
                    swal({
                        title:"标题不能为空",
                        icon:"error"
                    })
                    return ;
                }
                if(!content || !markdown){
                    swal({
                        title:"标题不能为空",
                        icon:"error"
                    })
                    return ;
                }
                axios.post("/api/Blog/article/create", {
                    _token: csrf_token,
                    markdown: markdown,
                    title: title,
                    content: content,
                    class_id: class_id
                }).then( r =>{
                    const data = r.data;

                    if(!data.success){
                        data.result.forEach(function(value){
                            iziToast.error({
                                title:"error",
                                message:value,
                                position:"topRight",
                                timeout:10000
                            })
                        })
                    }else{
                        iziToast.success({
                            title:"Success",
                            message:'发布成功!',
                            position:"topRight",
                            timeout:1500
                        })
                        localStorage.removeItem("blog_article_create_title")
                        localStorage.removeItem("blog_article_create_class_id")
                        this.vditor.clearCache()
                        setTimeout(() =>{
                            location.href=data.result.msg
                        },1500)
                    }
                }).catch(e=>{
                    iziToast.error({
                        title:"Error",
                        message:"请求出错,详细查看控制台",
                        position:"topRight",
                    })
                    console.error(e)
                })
            }
        },
        watch:{
            title:(title=>{
                localStorage.setItem("blog_article_create_title",title)
            }),
            class_id:(class_id=>{
                localStorage.setItem("blog_article_create_class_id",class_id)
            })
        }
    }
    Vue.createApp(app).mount('#vue-blog-article-create');
}

function isNumber(val){
    const regPos = /^[0-9]+.?[0-9]*/; //判断是否是数字。
    return regPos.test(val);
}



if(document.getElementById('vue-blog-article-edit')){
    const app = {
        data(){
            return {
                article_id:article_id,
                vditor:null,
                title:null,
                class_id:'none',
                classList:null
            }
        },
        mounted(){
            this.init()
        },
        methods:{
            init(){
                axios.post("/api/Blog/article/class",{_token:csrf_token})
                    .then(response => {
                        this.classList = response.data
                    })
                    .catch(e=>{
                        console.error(e)
                    })

                // vditor
                this.vditor = new Vditor('content-vditor', {
                    height: 400,
                    toolbarConfig: {
                        pin: true,
                    },
                    preview: {
                        markdown: {
                            toc: true,
                            mark: true,
                            autoSpace: true
                        }
                    },
                    toolbar: [
                        "emoji",
                        "headings",
                        "bold",
                        "italic",
                        "strike",
                        "link",
                        "|",
                        "list",
                        "ordered-list",
                        "|",
                        "line",
                        "insert-before",
                        "insert-after",
                        "table",
                        "|",
                        "fullscreen",
                        "edit-mode",

                    ],
                    counter: {
                        "enable": true,
                        "type": "已写字数"
                    },
                    after: () => {
                        axios.post("/api/Blog/open/article/data",{id:this.article_id})
                            .then(response => {
                                this.title = response.data.title
                                this.class_id = response.data.class_id
                                this.vditor.setValue(response.data.markdown)
                            })
                            .catch(e=>{
                                console.error(e)
                            })
                    },
                    typewriterMode: true,
                    placeholder: "请输入正文",
                });


            },
            submit(){
                const {title, class_id} = this;
                const content = this.vditor.getHTML();
                const markdown = this.vditor.getValue();
                if(!isNumber(class_id)){
                    swal({
                        title:"请选择有效的分类",
                        icon:"error"
                    })
                    return ;
                }
                if(!title){
                    swal({
                        title:"标题不能为空",
                        icon:"error"
                    })
                    return ;
                }
                if(!content || !markdown){
                    swal({
                        title:"标题不能为空",
                        icon:"error"
                    })
                    return ;
                }
                axios.post("/api/Blog/article/edit", {
                    id:this.article_id,
                    _token: csrf_token,
                    markdown: markdown,
                    title: title,
                    content: content,
                    class_id: class_id
                }).then( r =>{
                    const data = r.data;

                    if(!data.success){
                        data.result.forEach(function(value){
                            iziToast.error({
                                title:"error",
                                message:value,
                                position:"topRight",
                                timeout:10000
                            })
                        })
                    }else{
                        iziToast.success({
                            title:"Success",
                            message:'更新成功!',
                            position:"topRight",
                            timeout:1500
                        })
                        setTimeout(() =>{
                            location.href=data.result.msg
                        },1500)
                    }
                }).catch(e=>{
                    iziToast.error({
                        title:"Error",
                        message:"请求出错,详细查看控制台",
                        position:"topRight",
                    })
                    console.error(e)
                })
            }
        }
    }
    Vue.createApp(app).mount('#vue-blog-article-edit');
}