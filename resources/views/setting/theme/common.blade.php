<div class="card card-body">

    <div class="mb-3">
        <div class="form-label">Gavatar镜像源</div>
        <select v-model="data.theme_common_gavatar" class="form-select" >
            <option value="https://www.gravatar.com/avatar/">官方</option>
            <option value="https://cn.gravatar.com/avatar/">官方cn源</option>
            <option value="https://gravatar.loli.net/avatar/">loli.net</option>
            <option value="https://sdn.geekzu.org/avatar/">极客族</option>
            <option value="https://cdn.v2ex.com/gravatar/">V2EX</option>
            <option value="https://dn-qiniu-avatar.qbox.me/avatar/">七牛云</option>
        </select>
    </div>

    <div class="mb-3">
        <div class="form-label">网站icon</div>
        <input type="text" class="form-control" v-model="data.theme_common_icon">
        <small>填写链接,<a href="/admin/users/files/upload" target="_blank">点我</a> 上传文件</small>
    </div>

</div>