<div class="row row-cards">
    @foreach(Itf()->get('ui-index-header-hook') as $k=>$v)
        @if(call_user_func($value['enable'])===true)
            @include($value['view'])
        @endif
    @endforeach
</div>