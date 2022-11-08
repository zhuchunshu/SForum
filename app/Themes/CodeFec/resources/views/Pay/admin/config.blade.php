@extends("app")

@section('title',"支付配置")

@section('content')
    <form action="/admin/Pay/config" method="post" enctype="multipart/form-data">
        <x-csrf/>
        <div class="row row-cards">
            <div class="col-md-12">
                <!-- Cards with tabs component -->
                <div class="card-tabs border-0">
                    <!-- Cards navigation -->
                    <ul class="nav nav-tabs">
                        @foreach(pay()->getInterfaces() as $key=>$value)
                            @if($key==1)
                                <li class="nav-item"><a href="#{{$value['name']}}" class="nav-link active" data-bs-toggle="tab">{{$value['name']}}</a></li>
                            @else
                                <li class="nav-item"><a href="#{{$value['name']}}" class="nav-link" data-bs-toggle="tab">{{$value['name']}}</a></li>
                            @endif
                        @endforeach
                    </ul>
                    <div class="tab-content">
                        <!-- Content of card #1 -->
                        @foreach(pay()->getInterfaces() as $key=>$value)
                            @if($key==1)
                                <div id="{{$value['name']}}" class="border-0 card tab-pane active show">
                                    @include($value['view'])
                                </div>
                            @else
                                <div id="{{$value['name']}}" class="border-0 card tab-pane">
                                    @include($value['view'])
                                </div>
                            @endif
                        @endforeach
                    </div>
                </div>
            </div>
            <div class="col-md-12">
                <button @@click="submit" class="btn btn-light">提交</button>
            </div>
        </div>
    </form>
@endsection

@section('scripts')
    <script src="{{ mix("js/admin/pay.js") }}"></script>
@endsection