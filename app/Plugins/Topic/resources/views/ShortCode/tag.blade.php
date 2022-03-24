<div class="row row-cards justify-content-center">
    <div class="col-md-6">
        <div class="card">
            <div class="card-status-top" style="{{ Core_Ui()->Css()->bg_color($value->color) }}"></div>
            <div class="card-body">
                <div class="row">
                    <div class="col-auto">
                        <span class="avatar" style="background-image: url({{ $value->icon }})"></span>
                    </div>
                    <div class="col">
                        <a href="/tags/{{ $value->id }}.html"
                           class="card-title text-h2">{{ $value->name }}</a>
                        {{ \Hyperf\Utils\Str::limit(core_default($value->description, '暂无描述'), 32) }}
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>