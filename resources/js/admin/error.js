const { default: axios } = require("axios");

const empty = {
  data() {
    return {
      code: 200,
      msg: "error",
      url: "/",
    };
  },
  mounted() {
    // 获取错误信息
    axios
      .get(location.href + "?data=json")
      .then(response=>{
          this.code = response.data.code;
          this.msg = response.data.result.msg;
          if(response.data.result.back){
              this.url = response.data.result.back;
          }else{
              this.url = document.referrer;
          }
      })
      .catch(function (error) {
        console.log(error);
        swal({
          title: "错误信息获取失败! 详细请查看控制台",
        });
      });
    // 获取跳转链接
    axios.post("/api/AdminErrorRedirect",{path:location.pathname,_token:csrf_token}).then((response) => {
        var data = response.data;
        if(data.success===true){
            this.url = data.result.data
        }
    });
  },
};

Vue.createApp(empty).mount("#empty");
