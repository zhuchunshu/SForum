import axios from "axios";
import iziToast from "izitoast";
import swal from "sweetalert";

if(document.getElementById("report-data-card-footer")){
    const rdcf = {
        data(){
            return {
                btn:{
                    class:{
                        isPrimary:false,
                        isDanger:false,
                    },
                    text:"loading ..."
                }
            }
        },
        methods:{
            submit(){
                axios.post("/api/core/report/update",{
                    _token:csrf_token,
                    report_id:report_id
                }).then(r=>{
                    if(!r.data.success){
                        r.data.result.forEach(function(value){
                            iziToast.error({
                                title:"error",
                                message:value,
                                position:"topRight",
                                timeout:10000
                            })
                        })
                    }else{
                        r.data.result.forEach(function(value){
                            iziToast.success({
                                title:"Success",
                                message:value,
                                position:"topRight"
                            })
                        })
                        this.init();
                    }
                }).catch(e=>{
                    console.error(e)
                    iziToast.error({
                        position:"topRight",
                        title:"Error",
                        message:"请求出错,详细查看控制台"
                    })
                })
            },
            remove(){
                swal("确定要删除此举报吗? 删除后不可恢复", {
                    dangerMode: true,
                    buttons: true,
                }).then((value)=>{
                    if(value===true){
                        axios.post("/api/core/report/remove",{
                            _token:csrf_token,
                            report_id:report_id
                        }).then(r=>{
                            if(!r.data.success){
                                r.data.result.forEach(function(value){
                                    iziToast.error({
                                        title:"error",
                                        message:value,
                                        position:"topRight",
                                        timeout:10000
                                    })
                                })
                            }else{
                                r.data.result.forEach(function(value){
                                    iziToast.success({
                                        title:"Success",
                                        message:value,
                                        position:"topRight"
                                    })
                                })
                                location.href="/report"
                            }
                        }).catch(e=>{
                            console.error(e)
                            iziToast.error({
                                position:"topRight",
                                title:"Error",
                                message:"请求出错,详细查看控制台"
                            })
                        })
                    }
                });
            },
            init(){
                axios.post("/api/core/report/data", {
                    _token: csrf_token,
                    report_id:report_id
                }).then(r =>{
                    if(!r.data.success){
                        r.data.result.forEach(function(value){
                            iziToast.error({
                                title:"error",
                                message:value,
                                position:"topRight",
                                timeout:10000
                            })
                        })
                    }else{
                        var result = r.data.result;
                        if(result.status==="pending"){
                            this.btn.class.isPrimary=true
                            this.btn.class.isDanger=false
                            this.btn.text="批准";
                        }
                        if(result.status==="approve"){
                            this.btn.class.isPrimary=false
                            this.btn.class.isDanger=true
                            this.btn.text="驳回";
                        }
                        if(result.status==="reject"){
                            this.btn.class.isPrimary=true
                            this.btn.class.isDanger=false
                            this.btn.text="批准";
                        }
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
         },
        mounted(){
            this.init();
        }
    }

    Vue.createApp(rdcf).mount("#report-data-card-footer")
}