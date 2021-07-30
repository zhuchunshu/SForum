import axios from "axios"
import swal from 'sweetalert';

const form = {
    data() {
        return {
            username:'',
            password:'',
            csrf_token:''
        }
    },
    methods: {
        submit(){
            axios.post("/admin/login",{username:this.username,password:this.password,_token:csrf_token})
            .then(function(response){
                var data = response.data
                var content="";
                if(data.success===false){
                    if(data.result instanceof Array){
                        data.result.forEach(element => {
                            content = content+element+"\n"
                        });
                        swal({
                            icon : "error",
                            title: "出错啦!",
                            text:content
                        });
                    }
                }else{
                    // 登陆成功
                    swal({
                        icon : "success",
                        title: "Success",
                        text:   data.result.msg
                    });
                    setTimeout(() => {
                        location.href=data.result.url
                    }, 1500);
                }
            })
            .catch(function(error){
                swal({
                    icon : "error",
                    title: "请求错误,详细查看控制台输出",
                });
                console.log(error)
            })
        }
    },
}
Vue.createApp(form).mount("#form")