import axios from "axios";
import swal from "sweetalert";

if (document.getElementById("vue-topic-tag-table")) {
    const vttt = {
        methods: {
            remove(id) {
                axios.post("/admin/topic/tag/remove", {
                    _token: csrf_token,
                    id: id
                }).then(response => {
                    var data = response.data;
                    if (data.success) {
                        swal({
                            title: data.result.msg,
                            icon: "success"
                        })
                    } else {
                        swal({
                            title: data.result.msg,
                            icon: "error"
                        })
                    }
                }).catch(error => {
                    swal({
                        title: "请求出错,详细查看控制台",
                        icon: "error"
                    })
                    console.error(error)
                })
            }
        }
    }
    Vue.createApp(vttt).mount("#vue-topic-tag-table");
}

if (document.getElementById("vue-topic-tag-jobs-table")) {
    const app = {
        methods: {
            // 批准
            approval(id) {
                axios.post("/admin/topic/tag/job/approval", {
                    _token: csrf_token,
                    id: id
                }).then(response => {
                    var data = response.data;
                    if (data.success) {
                        swal({
                            title: data.result.msg,
                            icon: "success"
                        })
                    } else {
                        swal({
                            title: data.result.msg,
                            icon: "error"
                        })
                    }
                }).catch(error => {
                    swal({
                        title: "请求出错,详细查看控制台",
                        icon: "error"
                    })
                    console.error(error)
                })
            },
            // 驳回
            reject(id) {
                swal("驳回理由:", {
                    content: "input",
                    buttons: true
                })
                    .then((value) => {
                        if (value !== null) {
                            axios.post("/admin/topic/tag/job/reject", {
                                _token: csrf_token,
                                id: id,
                                content:value
                            }).then(response => {
                                var data = response.data;
                                if (data.success) {
                                    swal({
                                        title: data.result.msg,
                                        icon: "success"
                                    })
                                } else {
                                    swal({
                                        title: data.result.msg,
                                        icon: "error"
                                    })
                                }
                            }).catch(error => {
                                swal({
                                    title: "请求出错,详细查看控制台",
                                    icon: "error"
                                })
                                console.error(error)
                            })
                        }
                        //swal(`You typed: ${value}`);
                    });

            }
        }
    }
    Vue.createApp(app).mount("#vue-topic-tag-jobs-table");
}