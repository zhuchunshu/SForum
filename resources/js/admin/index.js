import axios from "axios";
var md = require('markdown-it')({
    html:         true,        // 在源码中启用 HTML 标签
    xhtmlOut:     true,        // 使用 '/' 来闭合单标签 （比如 <br />）。
                                // 这个选项只对完全的 CommonMark 模式兼容。
    breaks:       false,        // 转换段落里的 '\n' 到 <br>。
    langPrefix:   'language-',  // 给围栏代码块的 CSS 语言前缀。对于额外的高亮代码非常有用。
    linkify:      false,        // 将类似 URL 的文本自动转换为链接。

    // 启用一些语言中立的替换 + 引号美化
    typographer:  false,

    // 双 + 单引号替换对，当 typographer 启用时。
    // 或者智能引号等，可以是 String 或 Array。
    //
    // 比方说，你可以支持 '«»„“' 给俄罗斯人使用， '„“‚‘'  给德国人使用。
    // 还有 ['«\xA0', '\xA0»', '‹\xA0', '\xA0›'] 给法国人使用（包括 nbsp）。
    quotes: '“”‘’',

    // 高亮函数，会返回转义的HTML。
    // 或 '' 如果源字符串未更改，则应在外部进行转义。
    // 如果结果以 <pre ... 开头，内部包装器则会跳过。
    highlight: function (/*str, lang*/) { return ''; }
});

if(document.getElementById("vue-admin-index-releases")){
    const app = {
        data(){
            return {
                data:null,
                markdown:null,
                updateLog:null,
            }
        },
        mounted(){
            this.version()
        },
        methods:{
            // 初始化
            version(){
                axios.post("/api/admin/getVersion", {
                    _token:csrf_token
                }).then(r =>{
                    this.data = r.data;
                })
            },
            getUpdateLog(){
                axios.post("/api/admin/getUpdateLog",{
                    _token:csrf_token
                }).then(r=>{
                    this.markdown = r.data
                    this.updateLog=md.render(this.markdown)
                })
            },
            clearCache(){
                axios.post("/api/admin/clearCache",{
                    _token:csrf_token
                }).then(r=>{
                    const data =r.data
                    if(!data.success){
                        swal({
                            icon:"error",
                            title:"出错啦!",
                            content:data.result.msg
                        })
                    }else{
                        swal({
                            icon:"success",
                            title:"Success!",
                            content:data.result.msg
                        })
                    }
                })
            },
            update(){
                swal("安全起见，请手动更新","更新命令: php CodeFec CodeFec:Upgrading")
            }
        }
    }
    Vue.createApp(app).mount("#vue-admin-index-releases")
}


if(document.getElementById("vue-admin-panel-releases")){
    const app = {
        data(){
            return {
                data:null,
                author:null
            }
        },
        mounted(){
            this.init()
        },
        methods:{
            // 初始化
            init(){
                axios.post("/api/admin/getRelease/"+release_id, {
                    _token:csrf_token
                }).then(r =>{
                    this.data=r.data;
                    this.getAuthor()
                })
            },
            // get author
            getAuthor(){
                axios.get(this.data.author.url).then(r=>{
                    this.author = r.data;
                })
            }
        }
    }
    Vue.createApp(app).mount("#vue-admin-panel-releases")
}