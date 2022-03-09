import axios from "axios";
import iziToast from "izitoast";

if(document.getElementById("vue-blog-class-list")){
    const app = {
        data(){
            return {

            }
        },
        methods:{
            remove(token){
                if(confirm('确定要删除此分类吗? 此分类下的帖子也会被删除,删除后不可恢复')===true){
                    axios.post('/blog/class/remove', {
                        _token:csrf_token,
                        token:token
                    }).then(r=>{
                        const data = r.data;
                        if(data.success){
                            iziToast.success({
                                title:"success",
                                message:data.result.msg,
                                position:"topRight"
                            })
                            setTimeout(()=>{
                                location.reload()
                            },1000)
                        }else{
                            iziToast.error({
                                title:"error",
                                message:data.result.msg,
                                position:"topRight"
                            })
                        }
                    })
                }
            }
        }
    }
    Vue.createApp(app).mount("#vue-blog-class-list");
}