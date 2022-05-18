<div class="col-md-12 p-3">
    <div class="row row-cards" id="users-settings-form">
        <div class="col-md-12">
            <!-- Cards with tabs component -->
            @if(count(Itf()->get('users_options')))
                <div class="card-tabs border-0">
                    <!-- Cards navigation -->
                    <ul class="nav nav-tabs">
                        @foreach(Itf()->get('users_options') as $key=>$value)
                            @if(core_Itf_id('users_options',$key)==1)
                                <li class="nav-item"><a href="#{{$key}}" class="nav-link active" data-bs-toggle="tab">{{$value['name']}}</a></li>
                            @else
                                <li class="nav-item"><a href="#{{$key}}" class="nav-link" data-bs-toggle="tab">{{$value['name']}}</a></li>
                            @endif
                        @endforeach
                    </ul>
                    <div class="tab-content">
                        <!-- Content of card #1 -->
                        @foreach(Itf()->get('users_options') as $key=>$value)
                            @if(core_Itf_id('users_options',$key)==1)
                                <div id="{{$key}}" class="border-0 card tab-pane active show">
                                    @include($value['view'])
                                </div>
                            @else
                                <div id="{{$$key}}" class="border-0 card tab-pane">
                                    @include($value['view'])
                                </div>
                            @endif
                        @endforeach
                    </div>
                </div>
            @else
                <div class="alert alert-warning">
                    <i class="fa fa-warning"></i>
                    <strong>无可设置项</strong>
                </div>
            @endif
        </div>
        @if(count(Itf()->get('users_options')))
        <div class="p-3">
            <button @@click="submit" class="btn btn-light">提交</button>
        </div>
        @endif
    </div>
</div>