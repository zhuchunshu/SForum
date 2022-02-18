import axios from "axios";

if(document.getElementById("vue-admin-index-releases")){
    const app = {
        data(){
            return {
                data:null,
                commit:null,
            }
        },
        mounted(){
            this.version()
            this.commits()
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
            commits(){
                axios.post("/api/admin/getCommit",{
                    _token:csrf_token
                }).then(r=>{
                    this.commit = r.data
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