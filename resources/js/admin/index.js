import axios from "axios";
var md = require('markdown-it')();

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
            this.getUpdateLog()
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
                axios.post("/api/admin/update",{
                    _token:csrf_token
                }).then(r=>{
                    const data =r.data
                    if(!data.success){
                        swal({
                            icon:"error",
                            title:data.result.msg,
                        })
                    }else{
                        swal({
                            icon:"success",
                            title: data.result.msg,
                        })
                    }
                })
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