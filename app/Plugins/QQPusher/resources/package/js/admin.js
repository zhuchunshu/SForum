import axios from "axios";

Vue.createApp({
    data(){
        return {
            checkeds:[

            ],
            num: 0,
        }
    },
    mounted(){
        axios.get('/admin/QQPusher/groups')
            .then(r=>{
                this.checkeds = r.data.result.data
            })
    },
    watch:{
        checkeds: function (newval, oldval) {
            if (this.num <= 0) {
                this.num++;
            } else {
                axios
                    .post("/admin/QQPusher/groups", {
                        data: this.checkeds,
                        _token: csrf_token
                    })
                    .then(function (response) {
                        var data = response.data;
                        if (data.success === true) {
                            swal({
                                icon: "success",
                                title: data.result.msg,
                            });
                        } else {
                            swal({
                                icon: "error",
                                title: data.result.msg,
                            });
                        }
                    })
                    .catch(function (error) {
                        swal({
                            icon: "error",
                            title: "请求出错,详细查看控制台",
                        });
                        console.log(error);
                    });
            }
        },
    }
}).mount('#groups');