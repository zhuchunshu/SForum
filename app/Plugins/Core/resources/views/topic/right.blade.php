<div class="col-md-10">
    <div class="border-0 card">
        <div class="card-body">
            <div class="row">
                <div class="col-md-12">
                   <div class="row justify-content-center">
                       <span class="avatar avatar-lg center-block" style="background-image: url({{$data->tag->icon}})"></span>
                       <br>
                       <b class="card-title text-h2 text-center" style="margin-top: 5px;margin-bottom:2px">{{ $data->tag->name }}</b>
                       <span class="text-center" style="color:rgba(0,0,0,.45)">{{ \Hyperf\Utils\Str::limit(core_default($data->tag->description, '暂无描述'), 32) }}</span>
                       <br>
                       <a href="/tags/{{ $data->tag->id }}.html" class="btn btn-azure text-center">查看此标签</a>
                   </div>
                </div>
            </div>
        </div>
    </div>
</div>