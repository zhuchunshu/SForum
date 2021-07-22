<div id="common" class="card tab-pane">
    <div class="card card-body">
        <div class="mb-3">
            <label class="form-label">网站名称</label>
            <input type="text" min="1" max="3" class="form-control" v-model="data.web_name">
        </div>


        <div class="mb-3">
            <div class="form-label">是否导入mithril.js (默认导入mithril.js)</div>
            <select v-model="data.theme_common_require_mithril" class="form-select" >
                <option value="yes">导入</option>
                <option value="no">不导入</option>
            </select>
            <small>程序默认导入vue.js、alpine.js和mithril.js框架</small>
        </div>

        <div class="mb-3">
            <label class="form-label">网站描述</label>
            <textarea type="number" min="1" max="3" class="form-control" v-model="data.web_description"></textarea>
        </div>

    </div>
</div>

