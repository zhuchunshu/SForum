<h3 style="color: red">Ucloud SMS配置</h3>
<div class="row">
    <div class="col-2">
        <label class="form-label">PublicKey</label>
        <input type="text" v-model="data.sms_ucloud_publicKey" class="form-control">
    </div>
    <div class="col-2">
        <label class="form-label">PrivateKey</label>
        <input type="text" v-model="data.sms_ucloud_privateKey" class="form-control">
    </div>
    <div class="col-2">
        <label class="form-label">ProjectId</label>
        <input type="text" v-model="data.sms_ucloud_projectId" class="form-control">
        <small>项目ID。不填写为默认项目，子帐号必须填写。 参见 <a href="/api/summary/get_project_list">获取项目 ID</a></small>
    </div>
    <div class="col-3">
        <label class="form-label">短信签名</label>
        <input type="text" v-model="data.sms_ucloud_sign_name" class="form-control">
    </div>
    <div class="col-3">
        <label class="form-label">模板ID</label>
        <input type="text" v-model="data.sms_ucloud_template" class="form-control">
    </div>
</div>