<div id="common" class="card tab-pane">
    <div class="card card-body">
        <x-csrf/>
        <div class="mb-3">
            <label class="form-label">网站名称</label>
            <input type="number" min="1" max="3" class="form-control" v-model="data.web_name">
        </div>
        <div class="mb-3">
            <div class="form-label">主题</div>
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
            <div class="form-label">是否导入mithril.js (默认导入mithril.js)</div>
            <select v-model="data.theme_common_require_mithril" class="form-select" >
                <option value="yes">导入</option>
                <option value="no">不导入</option>
            </select>
            <small>程序默认导入vue.js和mithril.js框架</small>
        </div>

        <div class="mb-3">
            <label class="form-label">网站描述</label>
            <textarea type="number" min="1" max="3" class="form-control" v-model="data.web_description"></textarea>
        </div>

    </div>
</div>

