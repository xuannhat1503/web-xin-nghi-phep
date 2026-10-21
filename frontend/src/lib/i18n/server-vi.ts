// Vietnamese translations of the exact English messages returned by the Go API.
// Leave-form validation uses issue codes instead (see "issue.*" keys); this map covers everything else.
export const serverVi: Record<string, string> = {
	'"Other" must stay active and require a description.':
		'Lý do "Khác" phải luôn hoạt động và yêu cầu mô tả.',
	'A makeup workday must fall on a regular day off (e.g. Saturday or Sunday).':
		'Ngày làm bù phải rơi vào ngày nghỉ thường lệ (VD: Thứ 7 hoặc Chủ nhật).',
	'A working day must have at least 1 working hour.':
		'Một ngày làm việc phải có ít nhất 1 giờ làm.',
	"Add the year's holidays before confirming it.":
		'Hãy thêm các ngày nghỉ của năm trước khi xác nhận.',
	'Advance notice must be between 0 and 365 days.': 'Số ngày báo trước phải từ 0 đến 365.',
	'An employee cannot be their own manager.': 'Nhân viên không thể là Manager của chính mình.',
	'An unexpected error occurred': 'Đã có lỗi không mong muốn xảy ra.',
	'Another calendar entry already exists on this date.': 'Ngày này đã có một mục lịch khác.',
	'Another change is in progress. Please try again.':
		'Đang có thay đổi khác được xử lý. Vui lòng thử lại.',
	'At least one active administrator must remain.':
		'Phải còn ít nhất một quản trị viên đang hoạt động.',
	'At most 10 addresses.': 'Tối đa 10 địa chỉ.',
	'Change note must be at most 500 characters.': 'Ghi chú thay đổi tối đa 500 ký tự.',
	'Choose a valid type.': 'Vui lòng chọn loại hợp lệ.',
	'Choose working hours or clock hours.': 'Chọn tính theo giờ làm việc hoặc giờ đồng hồ.',
	'Code must be 2-40 characters: A-Z, 0-9 and _ (starting with a letter).':
		'Mã gồm 2-40 ký tự: A-Z, 0-9 và _ (bắt đầu bằng chữ cái).',
	'Could not create session': 'Không thể tạo phiên đăng nhập.',
	'Could not sign out': 'Không thể đăng xuất.',
	'End time must be after start time.': 'Giờ kết thúc phải sau giờ bắt đầu.',
	'Enter a time as HH:mm.': 'Nhập giờ theo định dạng HH:mm.',
	'Enter a valid date.': 'Nhập ngày hợp lệ.',
	'Enter a valid IANA time zone, e.g. Asia/Ho_Chi_Minh.':
		'Nhập múi giờ IANA hợp lệ, VD: Asia/Ho_Chi_Minh.',
	'Enter times as HH:mm in 5-minute steps.': 'Nhập giờ dạng HH:mm, bước 5 phút.',
	'Invalid identity': 'Danh tính không hợp lệ.',
	'Invalid or missing CSRF token': 'Phiên làm việc không hợp lệ. Hãy tải lại trang.',
	'Invalid request body': 'Dữ liệu gửi lên không hợp lệ.',
	'Label must be 2-200 characters.': 'Tên hiển thị gồm 2-200 ký tự.',
	'Choose MOCK, TEST_EMAIL or LIVE.': 'Chọn MOCK, TEST_EMAIL hoặc LIVE.',
	'LIVE is locked: set EMAIL_LIVE_ENABLED=true in the server .env and restart.':
		'LIVE đang bị khóa: đặt EMAIL_LIVE_ENABLED=true trong .env của server rồi khởi động lại.',
	'Configure SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD and SMTP_FROM in .env first.':
		'Cần cấu hình SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD và SMTP_FROM trong .env trước.',
	'Name must be 2-100 characters.': 'Tên gồm 2-100 ký tự.',
	'At least one leave type must stay active.': 'Phải còn ít nhất một loại nghỉ đang dùng.',
	'Leave unit must be 15, 30 or 60 minutes.': 'Bước thời gian phải là 15, 30 hoặc 60 phút.',
	'Lunch break must be within working hours.': 'Giờ nghỉ trưa phải nằm trong giờ làm việc.',
	'Lunch must end after it starts.': 'Giờ kết thúc nghỉ trưa phải sau giờ bắt đầu.',
	'Manager not found.': 'Không tìm thấy Manager.',
	'Method not allowed': 'Phương thức không được phép.',
	'Minimum leave cannot exceed one working day.':
		'Thời gian nghỉ tối thiểu không được vượt quá một ngày làm việc.',
	'Minimum leave must be a multiple of the leave unit.':
		'Thời gian nghỉ tối thiểu phải là bội số của bước thời gian.',
	'Name must be 2-200 characters.': 'Tên gồm 2-200 ký tự.',
	'Note must be at most 1000 characters.': 'Ghi chú tối đa 1000 ký tự.',
	'Number of reminders must be between 0 and 20.': 'Số lần nhắc phải từ 0 đến 20.',
	'Only company holidays can be part of a day.':
		'Chỉ ngày nghỉ của công ty mới được nghỉ một phần ngày.',
	'Only draft entries can be deleted. Revert it to draft first.':
		'Chỉ xóa được ngày nháp. Hãy chuyển về nháp trước.',
	'Past days can only be set when past dates are allowed.':
		'Chỉ đặt số ngày lùi khi cho phép chọn ngày đã qua.',
	'Past days must be between 0 and 365.': 'Số ngày lùi phải từ 0 đến 365.',
	'Reminder interval must be between 15 minutes and 7 days.':
		'Khoảng nhắc phải từ 15 phút đến 7 ngày.',
	'Request body is too large': 'Dữ liệu gửi lên quá lớn.',
	'Request origin is not allowed': 'Nguồn yêu cầu không được phép.',
	'Resource not found': 'Không tìm thấy dữ liệu.',
	'Role must be ADMIN or EMPLOYEE.': 'Vai trò phải là ADMIN hoặc EMPLOYEE.',
	'Rules for this leave type are required.': 'Cần thiết lập quy định cho loại nghỉ này.',
	'Select at least one working day.': 'Chọn ít nhất một ngày làm việc.',
	'Select valid working days.': 'Chọn ngày làm việc hợp lệ.',
	'Service temporarily unavailable': 'Dịch vụ tạm thời gián đoạn.',
	'Set both start and end time, or neither for a full day.':
		'Nhập cả giờ bắt đầu và kết thúc, hoặc để trống cả hai nếu nghỉ cả ngày.',
	'Set how many past days are allowed.': 'Nhập số ngày được phép lùi.',
	'Sign in required': 'Vui lòng đăng nhập.',
	'Sort order must be between 0 and 9999.': 'Thứ tự phải từ 0 đến 9999.',
	'Source must be at most 255 characters.': 'Nguồn tối đa 255 ký tự.',
	'The date must stay within the same calendar year.': 'Ngày phải thuộc cùng năm lịch.',
	'The manager must be an active employee.': 'Manager phải là nhân viên đang hoạt động.',
	'This assignment would create a reporting cycle.': 'Phân công này sẽ tạo vòng lặp quản lý.',
	'This calendar year has not been set up yet.': 'Năm lịch này chưa được thiết lập.',
	'This code already exists.': 'Mã này đã tồn tại.',
	'Too many requests, please try again later': 'Quá nhiều yêu cầu, vui lòng thử lại sau.',
	'Use 5-minute steps.': 'Dùng bước 5 phút.',
	'version query parameter is required': 'Thiếu thông tin phiên bản.',
	'Working hours must end after they start.': 'Giờ kết thúc làm việc phải sau giờ bắt đầu.',
	'You cannot deactivate or demote your own account.':
		'Bạn không thể tự vô hiệu hóa hoặc hạ quyền tài khoản của mình.',
	'You do not have access to this resource': 'Bạn không có quyền truy cập.',
	'You do not have access to this page.': 'Bạn không có quyền truy cập trang này.',
	'Your account is pending activation': 'Tài khoản của bạn đang chờ kích hoạt.',
	'This record was changed by someone else. Reload and try again.':
		'Dữ liệu đã được người khác thay đổi. Hãy tải lại và thử lại.',
	'Please correct the highlighted fields.': 'Vui lòng sửa các trường được đánh dấu.',
	'The service is temporarily unavailable.': 'Dịch vụ tạm thời gián đoạn.',
	'The service is temporarily unavailable. Please try again shortly.':
		'Dịch vụ tạm thời gián đoạn. Vui lòng thử lại sau ít phút.',
	'Something went wrong. Please try again.': 'Đã có lỗi xảy ra. Vui lòng thử lại.',
	'This request can no longer be edited.': 'Đơn này không còn chỉnh sửa được.',
	'This submission key was already used for a different request.':
		'Mã gửi đơn đã được dùng cho một đơn khác.',
	'Please fix the problems below before submitting.':
		'Vui lòng xử lý các vấn đề dưới đây trước khi gửi.',
	'A valid Idempotency-Key header is required.': 'Thiếu mã chống gửi trùng. Hãy tải lại trang.',
	'You already have 20 drafts. Submit or delete some first.':
		'Bạn đã có 20 bản nháp. Hãy gửi hoặc xóa bớt trước.',
	'Employee not found': 'Không tìm thấy nhân viên.',
	'Calendar day not found': 'Không tìm thấy ngày lịch.',
	'Leave request not found': 'Không tìm thấy đơn nghỉ.'
};
