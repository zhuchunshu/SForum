import axios from "axios";
import swal from "sweetalert";

if(document.getElementById("vue-topic-tag-table")){
    const vttt = {
        methods: {
            remove(id){
                axios.post("/admin/topic/tag/remove",{
                    _token:csrf_token,
                    id:id
                }).then(response => {
                    var data = response.data;
                    if(data.success) {
                        swal({
                            title: data.result.msg,
                            icon:"success"
                        })
                    }else{
                        swal({
                            title: data.result.msg,
                            icon:"error"
                        })
                    }
                }).catch(error => {
                    swal({
                        title:"请求出错,详细查看控制台",
                        icon:"error"
                    })
                    console.error(error)
                })
            }
        }
    }
    Vue.createApp(vttt).mount("#vue-topic-tag-table");
}