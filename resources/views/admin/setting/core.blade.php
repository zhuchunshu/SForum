@extends("app")

@section('title',"站点设置")

@section('content')
<div class="col-md-12">
    <div class="row row-cards" id="setting-core-form">
    <div class="col-md-12">
        <!-- Cards with tabs component -->
        <div class="card-tabs border-0">
            <!-- Cards navigation -->
            <ul class="nav nav-tabs">
                @foreach(Itf_Setting()->get() as $key=>$value)
                @if($key==1)
                <li class="nav-item"><a href="#{{$value['ename']}}" class="nav-link active" data-bs-toggle="tab">{{$value['name']}}</a></li>
                @else
                <li class="nav-item"><a href="#{{$value['ename']}}" class="nav-link" data-bs-toggle="tab">{{$value['name']}}</a></li>
                @endif    
                @endforeach
            </ul>
            <div class="tab-content" id="setting-core-form">
                <!-- Content of card #1 -->
                @foreach(Itf_Setting()->get() as $key=>$value)
                    @if($key==1)
                    <div id="{{$value['ename']}}" class="border-0 card tab-pane active show">
                        @include($value['view'])
                    </div>
                    @else
                    <div id="{{$value['ename']}}" class="border-0 card tab-pane">
                        @include($value['view'])
                    </div>
                    @endif
                @endforeach
            </div>
        </div>
    </div>
    <div class="col-md-12">
        <button @@click="submit" class="btn btn-light">提交</button>
        <button style="margin-left:5px" @@click="clearCache" class="btn btn-dark">清理缓存</button>
    </div>
    </div>
</div>
@endsection

@section('scripts')
    <script src="{{ mix("js/admin/setting.js") }}"></script>
@endsection