<div id="footer" class="card tab-pane">
    <div class="card card-body">
        <x-csrf/>
        <div class="mb-3">
            <label class="form-label">页脚风格</label>
            <input type="number" min="1" max="1" class="form-control" v-model="data.core_theme_footer">
            <small>填写以下风格ID,1-1</small>
        </div>
{{--        <div class="mb-3">--}}
{{--            <label class="form-label">页脚居右自定义代码</label>--}}
{{--            <textarea class="form-control" v-model="data.core_theme_footer_right_code"></textarea>--}}
{{--        </div>--}}
        <div class="row row-cards">
            <div class="col-md-12">
                <a class="link link-accent card-title">风格1</a>
                <iframe style="width: 100%" src="/help/core/viewRender/components.theme.footer-1.html"></iframe>
            </div>
{{--            <hr/>--}}
{{--            <div class="col-md-12">--}}
{{--                <a class="link link-accent card-title">风格2</a>--}}
{{--                <iframe style="width: 100%" src="/help/core/viewRender/components.theme.footer-4.html"></iframe>--}}
{{--            </div>--}}
        </div>
    </div>
</div>

