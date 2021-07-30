<div id="common" class="card tab-pane">
    <div class="card card-body">
        <div class="mb-3">
            <label class="form-label">网站名称</label>
            <input type="text" min="1" max="3" class="form-control" v-model="data.web_name">
        </div>

        <div class="mb-3">
            <div class="form-label">注册登录页主题</div>
            <select v-model="data.theme_common_theme" class="form-select" >
                <option value="light">light</option>
                <option value="dark">dark</option>
                <option value="cupcake">cupcake</option>
                <option value="bumblebee">bumblebee</option>
                <option value="emerald">emerald</option>
                <option value="corporate">corporate</option>
                <option value="synthwave">synthwave</option>
                <option value="retro">retro</option>
                <option value="cyberpunk">cyberpunk</option>
                <option value="valentine">valentine</option>
                <option value="halloween">halloween</option>
                <option value="garden">garden</option>
                <option value="forest">forest</option>
                <option value="aqua">aqua</option>
                <option value="lofi">lofi</option>
                <option value="pastel">pastel</option>
                <option value="fantasy">fantasy</option>
                <option value="wireframe">wireframe</option>
                <option value="black">black</option>
                <option value="luxury">luxury</option>
                <option value="dracula">dracula</option>
            </select>
        </div>

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
</div>

