<div id="header" class="card tab-pane active show">
    <div class="card card-body">
        <x-csrf/>
        <div class="mb-3">
            <label class="form-label">页头风格</label>
            <input type="number" min="1" max="4" class="form-control" v-model="data.core_theme_header">
            <small>填写以下风格ID,1-4</small>
        </div>
        <div class="row row-cards">
            <div class="col-md-12">
                <a class="link link-accent card-title">风格1</a>
                <iframe style="width: 100%" src="/help/core/viewRender/components.theme.header-1.html"></iframe>
            </div>
            <hr/>
            <div class="col-md-12">
                <a class="link link-accent card-title">风格2</a>
                <iframe style="width: 100%" src="/help/core/viewRender/components.theme.header-2.html"></iframe>
            </div>
            <hr/>
            <div class="col-md-12">
                <a class="link link-accent card-title">风格3</a>
                <iframe style="width: 100%" src="/help/core/viewRender/components.theme.header-3.html"></iframe>
            </div>
            <hr/>
            <div class="col-md-12">
                <a class="link link-accent card-title">风格4</a>
                <iframe style="width: 100%" src="/help/core/viewRender/components.theme.header-4.html"></iframe>
            </div>
        </div>
    </div>
</div>

