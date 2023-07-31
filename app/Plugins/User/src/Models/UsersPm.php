<?php

declare (strict_types=1);
namespace App\Plugins\User\src\Models;

use App\Model\Model;
use App\Plugins\Core\src\Models\Post;
use Carbon\Carbon;
/**
 * @property int $id 
 * @property string $post_id 
 * @property string $from_id 
 * @property string $to_id 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class UsersPm extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'users_pm';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected array $fillable = ['id', 'from_id', 'to_id', 'message', 'read', 'created_at', 'updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected array $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
    public function from_user()
    {
        return $this->belongsTo(User::class, "from_id", "id");
    }
    public function to_user()
    {
        return $this->belongsTo(User::class, "to_id", "id");
    }
}