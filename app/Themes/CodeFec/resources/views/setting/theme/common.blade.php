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
        <div class="form-label">是否导入mithril.js (默认导入mithril.js)</div>
        <select v-model="data.theme_common_require_mithril" class="form-select" >
            <option value="yes">导入</option>
            <option value="no">不导入</option>
        </select>
        <small>程序默认导入vue.js、alpine.js和mithril.js框架</small>
    </div>

</div>