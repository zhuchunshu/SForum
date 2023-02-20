<div class="row row-cards justify-content-center">
    <div class="col-md-10">
        <a href="/tags/{{$value->id}}.html" class="card card-link text-primary-fg" style="background-color: {{$value->color}}!important;">
            <div class="card-stamp">
                <div class="card-stamp-icon bg-yellow">
                    <!-- Download SVG icon from http://tabler-icons.io/i/star -->
                    {!! $value->icon !!}
                </div>
            </div>
            <div class="card-body">
                <h3 class="card-title">{{$value->name}}</h3>
                <p>{{ \Hyperf\Utils\Str::limit(core_default($value->description, __("app.no description")), 32) }}</p>
            </div>
        </a>
    </div>
</div>